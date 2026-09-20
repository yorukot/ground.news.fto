package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/yorukot/ground-news-tw/internal/config"
	"github.com/yorukot/ground-news-tw/internal/crawl"
	"github.com/yorukot/ground-news-tw/internal/db"
	"github.com/yorukot/ground-news-tw/internal/jobs"
	"github.com/yorukot/ground-news-tw/internal/seed"
)

func crawlCommand(ctx context.Context, cfg config.Config, command string, args []string) error {
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	slug := fs.String("outlet", "", "outlet slug")
	days := fs.Int("days", 7, "backfill days")
	asJSON := fs.Bool("json", false, "JSON report")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("unexpected positional argument")
	}
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	q := db.New(pool)
	if command == "crawl-sync" {
		if *slug == "" {
			return errors.New("crawl-sync requires --outlet slug")
		}
		for _, o := range seed.Outlets {
			if o.Slug == *slug {
				if o.Crawl == nil {
					return errors.New("outlet has no crawl configuration")
				}
				raw, err := json.Marshal(o.Crawl)
				if err != nil {
					return err
				}
				tag, err := pool.Exec(ctx, "UPDATE outlets SET crawl_config=$2 WHERE slug=$1", *slug, raw)
				if err != nil {
					return err
				}
				if tag.RowsAffected() == 0 {
					return errors.New("outlet not registered; run seed without --samples first")
				}
				return nil
			}
		}
		return fmt.Errorf("unknown outlet %q", *slug)
	}
	if command == "crawl-status" {
		stats, err := q.CrawlStatus(ctx, *slug)
		if err != nil {
			return err
		}
		failures, err := q.CrawlFailures(ctx, *slug)
		if err != nil {
			return err
		}
		sources, err := q.SourceStatus(ctx, *slug)
		if err != nil {
			return err
		}
		if *asJSON {
			sourceReports := make([]map[string]any, 0, len(sources))
			for _, s := range sources {
				sourceReports = append(sourceReports, map[string]any{"outlet": s.Slug, "mode": s.Mode, "source": s.Source, "progress": json.RawMessage(s.Progress), "error": s.Error})
			}
			return json.NewEncoder(os.Stdout).Encode(map[string]any{"outlets": stats, "failures": failures, "sources": sourceReports})
		}
		for _, s := range stats {
			fmt.Printf("%s: discovered=%d body=%d failed=%d waiting_ai=%d oldest=%v last_success=%v\n", s.Slug, s.Discovered, s.Fetched, s.Failed, s.AwaitingAnalysis, s.OldestWaiting, s.LastSuccess)
		}
		for _, f := range failures {
			fmt.Printf("  %s %s HTTP %d (%d): %s\n", f.Slug, f.Result, f.HttpStatus, f.Articles, f.Error)
		}
		for _, s := range sources {
			fmt.Printf("  %s %s %s progress=%s error=%s\n", s.Slug, s.Mode, s.Source, s.Progress, s.Error)
		}
		return nil
	}
	if *days < 1 || *days > 7 {
		return errors.New("days must be between 1 and 7")
	}
	outlets, err := q.ListEnabledOutlets(ctx)
	if err != nil {
		return err
	}
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{})
	if err != nil {
		return err
	}
	since := time.Now().UTC().Truncate(24 * time.Hour).Add(-time.Duration(*days) * 24 * time.Hour)
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		matched := false
		for _, o := range outlets {
			if *slug != "" && o.Slug != *slug {
				continue
			}
			matched = true
			// Seed missing-body URLs even when they disappeared from current feeds.
			_, err := tx.Exec(ctx, `INSERT INTO crawl_urls(url,outlet_id,headline,published_at,article_id)
			SELECT url,outlet_id,headline,published_at,id FROM articles WHERE outlet_id=$1 AND body='' AND published_at>=$2
			AND url NOT LIKE 'https://example.com/ground-sample/%' ON CONFLICT(url) DO UPDATE SET result='pending',error='',next_attempt_at=now()
			WHERE crawl_urls.result<>'success'`, o.ID, since)
			if err != nil {
				return err
			}
			if _, err := client.InsertTx(ctx, tx, jobs.CrawlOutletArgs{OutletID: o.ID, Since: &since}, nil); err != nil {
				return err
			}
		}
		if !matched {
			return errors.New("no matching enabled outlet")
		}
		return nil
	})
}

type checkResult struct {
	URL         string     `json:"url"`
	Headline    string     `json:"headline"`
	PublishedAt *time.Time `json:"publishedAt"`
	Extractor   string     `json:"extractor"`
	Length      int        `json:"length"`
	Start       string     `json:"start"`
	End         string     `json:"end"`
	Error       string     `json:"error,omitempty"`
}

func checkSources(ctx context.Context, cfg config.Config, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: app crawl-check slug [--count 10] [--json]")
	}
	slug := args[0]
	fs := flag.NewFlagSet("crawl-check", flag.ContinueOnError)
	count := fs.Int("count", 1, "articles to attempt")
	asJSON := fs.Bool("json", false, "JSON output")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if *count < 1 || *count > 100 {
		return errors.New("count must be 1..100")
	}
	var configRaw []byte
	for _, o := range seed.Outlets {
		if o.Slug == slug && o.Crawl != nil {
			configRaw, _ = json.Marshal(o.Crawl)
		}
	}
	if len(configRaw) == 0 {
		return fmt.Errorf("no configured outlet %q", slug)
	}
	c, pattern, err := crawl.ParseConfig(configRaw)
	if err != nil {
		return err
	}
	fetcher := crawl.NewFetcher(cfg.CrawlerUserAgent, hostDelay)
	items := []crawl.Found{}
	seen := map[string]bool{}
	sourceErrors := map[string]string{}
	for _, src := range c.Sources() {
		found, _, err := fetcher.DiscoverSource(ctx, c, src, crawl.Progress{}, 5)
		if err != nil {
			sourceErrors[src.URL] = err.Error()
			continue
		}
		for _, f := range found {
			if seen[f.URL] || (pattern != nil && !pattern.MatchString(f.URL)) {
				continue
			}
			seen[f.URL] = true
			items = append(items, f)
		}
	}
	results := []checkResult{}
	for _, item := range items[:min(len(items), *count)] {
		r := checkResult{URL: item.URL}
		body, final, err := fetcher.Get(ctx, item.URL)
		if err == nil {
			var a crawl.Article
			a, err = crawl.Extract(body, final, c.Selectors)
			if err == nil {
				r.Headline = a.Headline
				r.Extractor = a.Extractor
				if a.PublishedAt.IsZero() {
					a.PublishedAt = item.PublishedAt
				}
				if !a.PublishedAt.IsZero() {
					r.PublishedAt = &a.PublishedAt
				}
				text := []rune(a.Body)
				r.Length = len(text)
				r.Start = string(text[:min(120, len(text))])
				r.End = string(text[max(0, len(text)-120):])
			}
		}
		if err != nil {
			r.Error = err.Error()
		}
		results = append(results, r)
	}
	if *asJSON {
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"outlet": slug, "discovered": len(items), "sources": sourceErrors, "articles": results})
	}
	fmt.Printf("%s: discovered %d, attempted %d\n", slug, len(items), len(results))
	for source, err := range sourceErrors {
		fmt.Printf("source %s: %s\n", source, err)
	}
	for _, r := range results {
		fmt.Printf("%s\n%s (%s, %d runes) %s\nstart: %s\nend: %s\n", r.URL, r.Headline, r.Extractor, r.Length, r.Error, r.Start, r.End)
	}
	return nil
}
