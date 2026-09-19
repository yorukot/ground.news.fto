package jobs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/yorukot/ground-news-tw/internal/crawl"
	"github.com/yorukot/ground-news-tw/internal/db"
	"github.com/yorukot/ground-news-tw/internal/dedupe"
)

// CrawlAllArgs fans out one crawl per enabled outlet.
type CrawlAllArgs struct{}

func (CrawlAllArgs) Kind() string { return "crawl_all" }

func (CrawlAllArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueCrawl, UniqueOpts: river.UniqueOpts{ByPeriod: crawlInterval}}
}

type CrawlAllWorker struct {
	river.WorkerDefaults[CrawlAllArgs]
	deps Deps
}

func (w *CrawlAllWorker) Work(ctx context.Context, _ *river.Job[CrawlAllArgs]) error {
	outlets, err := db.New(w.deps.Pool).ListEnabledOutlets(ctx)
	if err != nil {
		return fmt.Errorf("list outlets: %w", err)
	}
	client := river.ClientFromContext[pgx.Tx](ctx)
	for _, o := range outlets {
		if _, err := client.Insert(ctx, CrawlOutletArgs{OutletID: o.ID}, nil); err != nil {
			return fmt.Errorf("enqueue crawl of %s: %w", o.Slug, err)
		}
	}
	return nil
}

type CrawlOutletArgs struct {
	OutletID int64 `json:"outlet_id" river:"unique"`
}

func (CrawlOutletArgs) Kind() string { return "crawl_outlet" }

func (CrawlOutletArgs) InsertOpts() river.InsertOpts {
	// Unique per outlet while one is still queued or running, so a slow site
	// never accumulates a backlog of crawls.
	return river.InsertOpts{Queue: QueueCrawl, MaxAttempts: 2, UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: crawlInterval}}
}

type CrawlOutletWorker struct {
	river.WorkerDefaults[CrawlOutletArgs]
	deps Deps
}

func (w *CrawlOutletWorker) Timeout(*river.Job[CrawlOutletArgs]) time.Duration {
	return 4 * time.Minute
}

func (w *CrawlOutletWorker) Work(ctx context.Context, job *river.Job[CrawlOutletArgs]) error {
	q := db.New(w.deps.Pool)
	outlet, err := q.GetOutlet(ctx, job.Args.OutletID)
	if err != nil {
		return fmt.Errorf("load outlet: %w", err)
	}
	if !outlet.Enabled {
		return nil
	}
	cfg, pattern, err := crawl.ParseConfig(outlet.CrawlConfig)
	if err != nil {
		return river.JobCancel(fmt.Errorf("outlet %s: %w", outlet.Slug, err))
	}

	purpose := crawl.ForAI
	if outlet.Coverage == coverageHeadline {
		purpose = crawl.ForIndex
	}
	found := w.deps.Fetcher.Discover(ctx, cfg, pattern, purpose)
	if len(found) == 0 {
		slog.Warn("crawl found nothing", "outlet", outlet.Slug)
		return nil
	}
	if outlet.Coverage == coverageHeadline {
		stored, err := storeHeadlines(ctx, w.deps, outlet.ID, found)
		slog.Info("crawled outlet", "outlet", outlet.Slug, "coverage", "headline", "found", len(found), "new", stored)
		return err
	}

	urls := make([]string, len(found))
	for i, f := range found {
		urls[i] = f.URL
	}
	existing, err := q.ExistingArticleURLs(ctx, urls)
	if err != nil {
		return fmt.Errorf("check existing urls: %w", err)
	}
	known := make(map[string]bool, len(existing))
	for _, u := range existing {
		known[u] = true
	}

	client := river.ClientFromContext[pgx.Tx](ctx)
	enqueued := 0
	for _, f := range found {
		if known[f.URL] || enqueued >= w.deps.MaxNewPerCrawl {
			continue
		}
		if !f.PublishedAt.IsZero() && time.Since(f.PublishedAt) > maxArticleAge {
			continue
		}
		args := FetchArticleArgs{OutletID: outlet.ID, URL: f.URL, FeedImageURL: f.ImageURL}
		if !f.PublishedAt.IsZero() {
			args.FeedPublishedAt = &f.PublishedAt
		}
		if _, err := client.Insert(ctx, args, nil); err != nil {
			return fmt.Errorf("enqueue fetch: %w", err)
		}
		enqueued++
	}
	slog.Info("crawled outlet", "outlet", outlet.Slug, "found", len(found), "new", enqueued)
	return nil
}

type FetchArticleArgs struct {
	OutletID int64  `json:"outlet_id"`
	URL      string `json:"url" river:"unique"`
	// FeedPublishedAt is the feed's date, used when the page itself has none.
	FeedPublishedAt *time.Time `json:"feed_published_at,omitempty"`
	FeedImageURL    string     `json:"feed_image_url,omitempty"`
}

func (FetchArticleArgs) Kind() string { return "fetch_article" }

func (FetchArticleArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueFetch, MaxAttempts: 4, UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

type FetchArticleWorker struct {
	river.WorkerDefaults[FetchArticleArgs]
	deps Deps
}

func (w *FetchArticleWorker) Timeout(*river.Job[FetchArticleArgs]) time.Duration {
	return 3 * time.Minute
}

func (w *FetchArticleWorker) Work(ctx context.Context, job *river.Job[FetchArticleArgs]) error {
	q := db.New(w.deps.Pool)
	outlet, err := q.GetOutlet(ctx, job.Args.OutletID)
	if err != nil {
		return fmt.Errorf("load outlet: %w", err)
	}
	cfg, _, err := crawl.ParseConfig(outlet.CrawlConfig)
	if err != nil {
		return river.JobCancel(err)
	}

	if outlet.Coverage == coverageHeadline {
		// Guards the policy even if a fetch job was enqueued before an
		// outlet's coverage changed: its pages must never be fetched for AI.
		return river.JobCancel(fmt.Errorf("outlet %s is headline-only", outlet.Slug))
	}

	page, finalURL, err := w.deps.Fetcher.Get(ctx, job.Args.URL)
	if errors.Is(err, crawl.ErrDisallowed) {
		return river.JobCancel(err)
	}
	if err != nil {
		return err
	}
	art, err := crawl.Extract(page, finalURL, cfg.Selectors)
	if errors.Is(err, crawl.ErrNoArticle) {
		return river.JobCancel(fmt.Errorf("%s: %w", job.Args.URL, err))
	}
	if err != nil {
		return err
	}
	if art.ImageURL == "" {
		art.ImageURL = job.Args.FeedImageURL
	}

	outletID := outlet.ID
	if outlet.IsAggregator {
		// An aggregator's article is credited to its original outlet. If that
		// outlet isn't one we cover, the article is not ours to show.
		outletID, err = q.FindOutletByName(ctx, art.Provider)
		if errors.Is(err, pgx.ErrNoRows) {
			return river.JobCancel(fmt.Errorf("aggregator article from uncovered publisher %q", art.Provider))
		}
		if err != nil {
			return fmt.Errorf("resolve publisher: %w", err)
		}
	}

	publishedAt := art.PublishedAt
	if publishedAt.IsZero() && job.Args.FeedPublishedAt != nil {
		publishedAt = *job.Args.FeedPublishedAt
	}
	if publishedAt.IsZero() {
		publishedAt = time.Now()
	}
	if time.Since(publishedAt) > maxArticleAge {
		return river.JobCancel(fmt.Errorf("%s: published %s, too old", job.Args.URL, publishedAt.Format(time.DateOnly)))
	}

	sum := sha256.Sum256([]byte(art.Body))
	return pgx.BeginFunc(ctx, w.deps.Pool, func(tx pgx.Tx) error {
		id, err := db.New(tx).InsertFetchedArticle(ctx, db.InsertFetchedArticleParams{
			OutletID:    outletID,
			Url:         job.Args.URL,
			Headline:    art.Headline,
			ImageUrl:    art.ImageURL,
			Body:        art.Body,
			PublishedAt: publishedAt,
			ContentHash: hex.EncodeToString(sum[:]),
			Minhash:     orEmpty(dedupe.Sign(art.Body)),
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // already stored by an earlier attempt
		}
		if err != nil {
			return fmt.Errorf("store article: %w", err)
		}
		_, err = river.ClientFromContext[pgx.Tx](ctx).InsertTx(ctx, tx, DedupeArticleArgs{ArticleID: id}, nil)
		return err
	})
}

func orEmpty(sig dedupe.Signature) []int32 {
	if sig == nil {
		return []int32{}
	}
	return sig
}
