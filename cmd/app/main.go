// Command app is the single backend binary: serve, worker, migrate and seed.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/yorukot/ground-news-tw/internal/admin"
	"github.com/yorukot/ground-news-tw/internal/ai"
	"github.com/yorukot/ground-news-tw/internal/config"
	"github.com/yorukot/ground-news-tw/internal/crawl"
	"github.com/yorukot/ground-news-tw/internal/db"
	"github.com/yorukot/ground-news-tw/internal/httpapi"
	"github.com/yorukot/ground-news-tw/internal/jobs"
	"github.com/yorukot/ground-news-tw/internal/seed"
)

// hostDelay is the minimum gap between two requests to one outlet's site.
const hostDelay = 3 * time.Second

const usage = `usage: app <command>

commands:
  serve     run the HTTP API
  worker    run the ingestion pipeline (crawl, dedupe, summarize, link)
  migrate   apply database migrations
  seed      register outlets; add --samples for fictional sample events
  crawl-check <outlet-slug>
            discover an outlet's feeds and extract one article, without
            storing anything; use it before enabling an outlet
`

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Args[1], os.Args[2:]); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, command string, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	switch command {
	case "migrate":
		return db.Migrate(ctx, cfg.DatabaseURL)
	case "seed":
		fs := flag.NewFlagSet("seed", flag.ExitOnError)
		samples := fs.Bool("samples", false, "also insert fictional sample events (development only)")
		if err := fs.Parse(args); err != nil {
			return err
		}
		pool, err := db.NewPool(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()
		return seed.Run(ctx, pool, *samples)
	case "serve":
		return serve(ctx, cfg)
	case "crawl-check":
		if len(args) != 1 {
			return fmt.Errorf("usage: app crawl-check <outlet-slug>")
		}
		return crawlCheck(ctx, cfg, args[0])
	case "worker":
		return work(ctx, cfg)
	default:
		fmt.Fprint(os.Stderr, usage)
		return fmt.Errorf("unknown command %q", command)
	}
}

// crawlCheck runs discovery and one extraction for an outlet from the seed
// list, through the same fetcher the worker uses (robots.txt included).
func crawlCheck(ctx context.Context, cfg config.Config, slug string) error {
	var outlet *seed.Outlet
	for i := range seed.Outlets {
		if seed.Outlets[i].Slug == slug {
			outlet = &seed.Outlets[i]
		}
	}
	if outlet == nil {
		return fmt.Errorf("unknown outlet %q", slug)
	}
	raw, err := json.Marshal(outlet.Crawl)
	if err != nil {
		return err
	}
	crawlCfg, pattern, err := crawl.ParseConfig(raw)
	if err != nil {
		return err
	}

	fetcher := crawl.NewFetcher(cfg.CrawlerUserAgent, hostDelay)
	purpose := crawl.ForAI
	if outlet.Coverage == seed.CoverageHeadline {
		purpose = crawl.ForIndex
	}
	found := fetcher.Discover(ctx, crawlCfg, pattern, purpose)
	fmt.Printf("%s: %d article urls discovered\n", slug, len(found))
	if outlet.Coverage == seed.CoverageHeadline {
		titled := 0
		for i, f := range found {
			if f.Title != "" && !f.PublishedAt.IsZero() {
				titled++
			}
			if i < 3 {
				fmt.Printf("  %s | %s | %s\n", f.PublishedAt.Format(time.RFC3339), f.Title, f.URL)
			}
		}
		fmt.Printf("  headline-only: %d of %d have a title and a date; no article page is fetched\n", titled, len(found))
		return nil
	}
	for _, f := range found {
		page, final, err := fetcher.Get(ctx, f.URL)
		if err != nil {
			fmt.Printf("  %s\n  fetch failed: %v\n", f.URL, err)
			if errors.Is(err, crawl.ErrDisallowed) {
				return err
			}
			continue
		}
		art, err := crawl.Extract(page, final, crawlCfg.Selectors)
		if err != nil {
			fmt.Printf("  %s\n  extract failed: %v\n", f.URL, err)
			continue
		}
		body := []rune(art.Body)
		fmt.Printf("  url:       %s\n  headline:  %s\n  published: %s (feed: %s)\n  provider:  %s\n  body:      %d runes\n  start:     %s\n  end:       %s\n",
			f.URL, art.Headline, art.PublishedAt.Format(time.RFC3339), f.PublishedAt.Format(time.RFC3339), art.Provider,
			len(body), string(body[:min(90, len(body))]), string(body[max(0, len(body)-70):]))
		return nil
	}
	return errors.New("no article could be extracted")
}

func work(ctx context.Context, cfg config.Config) error {
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	var model ai.Client = ai.Fake{}
	if cfg.OpenAIAPIKey != "" {
		model = ai.NewOpenAI(cfg.OpenAIAPIKey, cfg.OpenAIBaseURL, cfg.OpenAISummaryModel)
	} else {
		slog.Warn("OPENAI_API_KEY is not set; using the fake AI client")
	}

	deps := jobs.Deps{
		Pool:           pool,
		AI:             model,
		Fetcher:        crawl.NewFetcher(cfg.CrawlerUserAgent, hostDelay),
		MaxNewPerCrawl: cfg.CrawlMaxNewPerRun,
	}
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues:       jobs.Queues(),
		Workers:      jobs.Workers(deps),
		PeriodicJobs: jobs.PeriodicJobs(),
		Logger:       slog.Default(),
	})
	if err != nil {
		return fmt.Errorf("river client: %w", err)
	}
	if err := client.Start(ctx); err != nil {
		return fmt.Errorf("start worker: %w", err)
	}
	slog.Info("worker started", "model", model.Model())

	<-ctx.Done()
	// Let running jobs finish; River retries anything cut off.
	stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return client.Stop(stopCtx)
}

func serve(ctx context.Context, cfg config.Config) error {
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	queries := db.New(pool)
	var adminAPI *httpapi.Admin
	if cfg.AdminToken != "" {
		adminAPI = &httpapi.Admin{Token: cfg.AdminToken, Store: queries, Actions: &admin.Service{Pool: pool}}
	} else {
		slog.Info("ADMIN_TOKEN is not set; admin endpoints are disabled")
	}

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.New(queries, adminAPI, slog.Default()).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("api listening", "addr", cfg.HTTPAddr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
