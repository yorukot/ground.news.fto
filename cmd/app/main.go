// Command app is the single backend binary: serve, worker, migrate and seed.
package main

import (
	"context"
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
  crawl-check <slug> [--count 10] [--json]
            inspect discovery and extraction without storing or calling AI
  crawl-status [--outlet slug] [--json]
            report discovery, extraction failures and the AI backlog
  crawl-sync --outlet slug
            apply the checked-in source settings to one registered outlet
  crawl-backfill [--days 7] [--outlet slug]
            enqueue resumable discovery and missing-body recovery
  backfill-images
            fetch source image metadata for existing articles
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
	case "backfill-images":
		fs := flag.NewFlagSet("backfill-images", flag.ExitOnError)
		limit := fs.Int("limit", 200, "maximum number of articles to inspect")
		if err := fs.Parse(args); err != nil {
			return err
		}
		if *limit < 1 {
			return errors.New("limit must be positive")
		}
		return backfillImages(ctx, cfg, *limit)
	case "crawl-check":
		return checkSources(ctx, cfg, args)
	case "crawl-status", "crawl-backfill", "crawl-sync":
		return crawlCommand(ctx, cfg, command, args)
	case "worker":
		return work(ctx, cfg)
	default:
		fmt.Fprint(os.Stderr, usage)
		return fmt.Errorf("unknown command %q", command)
	}
}

// backfillImages stores publisher-advertised image URLs for articles that
// predate the image column. It fetches pages for indexing only, respects
// robots.txt, and never extracts or stores article text.
func backfillImages(ctx context.Context, cfg config.Config, limit int) error {
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	q := db.New(pool)
	articles, err := q.ListArticlesMissingImage(ctx, int32(limit))
	if err != nil {
		return fmt.Errorf("list articles missing images: %w", err)
	}
	fetcher := crawl.NewFetcher(cfg.CrawlerUserAgent, hostDelay)
	stored := 0
	for _, article := range articles {
		page, finalURL, err := fetcher.Get(ctx, article.Url)
		if err != nil {
			slog.Warn("image metadata fetch failed", "url", article.Url, "error", err)
			continue
		}
		imageURL := crawl.ExtractImage(page, finalURL)
		if imageURL == "" {
			continue
		}
		if err := q.SetArticleImage(ctx, db.SetArticleImageParams{ID: article.ID, ImageUrl: imageURL}); err != nil {
			return fmt.Errorf("store image for article %d: %w", article.ID, err)
		}
		stored++
	}
	slog.Info("image backfill complete", "inspected", len(articles), "stored", stored)
	return nil
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
		slog.Warn("OPENAI_API_KEY is not set; AI dispatch disabled unless ALLOW_FAKE_AI=true")
	}

	deps := jobs.Deps{
		Pool:               pool,
		AI:                 model,
		Fetcher:            crawl.NewFetcher(cfg.CrawlerUserAgent, hostDelay),
		DailyAnalysisLimit: cfg.DailyAnalysisLimit,
		AIEnabled:          cfg.OpenAIAPIKey != "" || cfg.AllowFakeAI,
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
