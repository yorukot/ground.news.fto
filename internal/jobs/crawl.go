package jobs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"
	"github.com/yorukot/ground-news-tw/internal/crawl"
	"github.com/yorukot/ground-news-tw/internal/db"
	"github.com/yorukot/ground-news-tw/internal/dedupe"
)

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
		return err
	}
	client := river.ClientFromContext[pgx.Tx](ctx)
	for _, o := range outlets {
		if _, err := client.Insert(ctx, CrawlOutletArgs{OutletID: o.ID}, nil); err != nil {
			return err
		}
	}
	_, err = client.Insert(ctx, DispatchFetchArgs{}, nil)
	return err
}

type CrawlOutletArgs struct {
	OutletID int64      `json:"outlet_id" river:"unique"`
	Since    *time.Time `json:"since,omitempty" river:"unique"`
}

func (CrawlOutletArgs) Kind() string { return "crawl_outlet" }
func (CrawlOutletArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueCrawl, MaxAttempts: 2, UniqueOpts: activeUnique()}
}

type CrawlOutletWorker struct {
	river.WorkerDefaults[CrawlOutletArgs]
	deps Deps
}

func (w *CrawlOutletWorker) Timeout(*river.Job[CrawlOutletArgs]) time.Duration {
	return 15 * time.Minute
}
func (w *CrawlOutletWorker) Work(ctx context.Context, job *river.Job[CrawlOutletArgs]) error {
	more, err := DiscoverOutlet(ctx, w.deps, job.Args.OutletID, job.Args.Since)
	if err != nil {
		return err
	}
	if job.Args.Since != nil && more {
		return river.JobSnooze(time.Minute)
	}
	return nil
}

// DiscoverOutlet commits URLs and source progress together. Failure never advances
// a cursor past URLs that have not been saved.
func DiscoverOutlet(ctx context.Context, d Deps, outletID int64, since *time.Time) (bool, error) {
	q := db.New(d.Pool)
	o, err := q.GetOutlet(ctx, outletID)
	if err != nil {
		return false, err
	}
	if !o.Enabled {
		return false, nil
	}
	cfg, pattern, err := crawl.ParseConfig(o.CrawlConfig)
	if err != nil {
		return false, err
	}
	mode, limit, cutoff := "live", 5, time.Now().Add(-maxArticleAge)
	if since != nil {
		mode, limit, cutoff = "backfill", 20, *since
	}
	more := false
	var failures []error
	for _, src := range cfg.Sources() {
		var progress crawl.Progress
		previous, e := q.GetSourceProgress(ctx, db.GetSourceProgressParams{OutletID: o.ID, Source: src.URL, Mode: mode})
		if e == nil {
			if err := json.Unmarshal(previous.Progress, &progress); err != nil {
				return false, err
			}
			if mode == "live" && progress.Done {
				progress = crawl.Progress{}
			}
			if mode == "backfill" && !previous.SinceAt.Equal(cutoff) {
				progress = crawl.Progress{}
			}
		} else if !errors.Is(e, pgx.ErrNoRows) {
			return false, e
		}
		progress.Since = cutoff
		batchLimit := limit
		if src.Kind == "sitemap" {
			batchLimit = 10
		}
		var fresh []crawl.Found
		var freshErr error
		// Refresh the front page even while walking an older saved cursor.
		if mode == "live" && !progress.Done && len(progress.Visited) > 0 {
			headLimit := 1
			if src.Kind == "sitemap" {
				headLimit = 2
			}
			fresh, _, freshErr = d.Fetcher.DiscoverSource(ctx, cfg, src, crawl.Progress{Since: cutoff}, headLimit)
			batchLimit -= headLimit
		}
		items, next, e := d.Fetcher.DiscoverSource(ctx, cfg, src, progress, batchLimit)
		items = append(fresh, items...)
		e = errors.Join(e, freshErr)
		errorText := ""
		if e != nil {
			errorText = e.Error()
			failures = append(failures, e)
		}
		encoded, _ := json.Marshal(next)
		err = pgx.BeginFunc(ctx, d.Pool, func(tx pgx.Tx) error {
			q := db.New(tx)
			for _, item := range items {
				if pattern != nil && !pattern.MatchString(item.URL) {
					continue
				}
				if !item.PublishedAt.IsZero() && item.PublishedAt.Before(cutoff) {
					continue
				}
				eligible, reason := crawl.InScope(item.Category)
				if err := q.DiscoverURL(ctx, db.DiscoverURLParams{Url: item.URL, OutletID: o.ID, Source: src.URL, Headline: item.Title, ImageUrl: item.ImageURL, Category: item.Category, Eligible: eligible, ScopeReason: reason, PublishedAt: timestamp(item.PublishedAt)}); err != nil {
					return err
				}
			}
			return q.SaveSourceProgress(ctx, db.SaveSourceProgressParams{OutletID: o.ID, Source: src.URL, Mode: mode, SinceAt: cutoff, Progress: encoded, Error: errorText})
		})
		if err != nil {
			return false, err
		}
		if !next.Done {
			more = true
		}
		slog.Info("discovered source", "outlet", o.Slug, "source", src.URL, "found", len(items), "complete", next.Done, "error", errorText)
	}
	return more, errors.Join(failures...)
}

type DispatchFetchArgs struct{}

func (DispatchFetchArgs) Kind() string { return "dispatch_fetch" }
func (DispatchFetchArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{UniqueOpts: river.UniqueOpts{ByPeriod: time.Minute}}
}

type DispatchFetchWorker struct {
	river.WorkerDefaults[DispatchFetchArgs]
	deps Deps
}

func (w *DispatchFetchWorker) Work(ctx context.Context, _ *river.Job[DispatchFetchArgs]) error {
	return pgx.BeginFunc(ctx, w.deps.Pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		items, err := q.ListFetchCandidates(ctx, 100)
		if err != nil {
			return err
		}
		for _, item := range items {
			args := FetchArticleArgs{OutletID: item.OutletID, URL: item.Url, FeedImageURL: item.ImageUrl}
			if item.PublishedAt.Valid {
				args.FeedPublishedAt = &item.PublishedAt.Time
			}
			if _, err := river.ClientFromContext[pgx.Tx](ctx).InsertTx(ctx, tx, args, nil); err != nil {
				return err
			}
			if err := q.ReserveFetch(ctx, item.Url); err != nil {
				return err
			}
		}
		return nil
	})
}

type FetchArticleArgs struct {
	OutletID        int64      `json:"outlet_id"`
	URL             string     `json:"url" river:"unique"`
	FeedPublishedAt *time.Time `json:"feed_published_at,omitempty"`
	FeedImageURL    string     `json:"feed_image_url,omitempty"`
}

func (FetchArticleArgs) Kind() string { return "fetch_article" }
func (FetchArticleArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueFetch, MaxAttempts: 4, UniqueOpts: activeUnique()}
}

type FetchArticleWorker struct {
	river.WorkerDefaults[FetchArticleArgs]
	deps Deps
}

func (w *FetchArticleWorker) Timeout(*river.Job[FetchArticleArgs]) time.Duration {
	return 5 * time.Minute
}
func (w *FetchArticleWorker) Work(ctx context.Context, job *river.Job[FetchArticleArgs]) error {
	q := db.New(w.deps.Pool)
	o, err := q.GetOutlet(ctx, job.Args.OutletID)
	if err != nil {
		return err
	}
	if !o.Enabled {
		return nil
	}
	cfg, _, err := crawl.ParseConfig(o.CrawlConfig)
	if err != nil {
		return river.JobCancel(err)
	}
	record, err := q.GetCrawlURL(ctx, job.Args.URL)
	if errors.Is(err, pgx.ErrNoRows) {
		var at time.Time
		if job.Args.FeedPublishedAt != nil {
			at = *job.Args.FeedPublishedAt
		}
		if err := q.DiscoverURL(ctx, db.DiscoverURLParams{Url: job.Args.URL, OutletID: o.ID, ImageUrl: job.Args.FeedImageURL, Eligible: true, ScopeReason: "legacy job", PublishedAt: timestamp(at)}); err != nil {
			return err
		}
		record, err = q.GetCrawlURL(ctx, job.Args.URL)
	}
	if err != nil {
		return err
	}
	if record.Result == "success" || !record.Eligible {
		return nil
	}
	page, final, err := w.deps.Fetcher.Get(ctx, job.Args.URL)
	if err != nil {
		return w.failure(ctx, job, err)
	}
	art, err := crawl.Extract(page, final, cfg.Selectors)
	if err != nil {
		return w.failure(ctx, job, err)
	}
	if art.ImageURL == "" {
		art.ImageURL = record.ImageUrl
	}
	if art.PublishedAt.IsZero() && record.PublishedAt.Valid {
		art.PublishedAt = record.PublishedAt.Time
	}
	if art.Category == "" {
		art.Category = record.Category
	}
	eligible, reason := crawl.InScope(art.Category)
	// An undated discovery can turn out to be an archive page. Compare with
	// discovery time, not execution time, so a queue delay never expires news.
	if !art.PublishedAt.IsZero() && art.PublishedAt.Before(record.FirstSeenAt.Add(-maxArticleAge)) {
		eligible, reason = false, "older than discovery window"
	}
	sum := sha256.Sum256([]byte(art.Body))
	return pgx.BeginFunc(ctx, w.deps.Pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		id, err := q.InsertFetchedArticle(ctx, db.InsertFetchedArticleParams{
			OutletID:    o.ID,
			Url:         job.Args.URL,
			Headline:    art.Headline,
			ImageUrl:    art.ImageURL,
			Body:        art.Body,
			PublishedAt: art.PublishedAt,
			ContentHash: hex.EncodeToString(sum[:]),
			Minhash:     orEmpty(dedupe.Sign(art.Body)),
		})
		if errors.Is(err, pgx.ErrNoRows) {
			var existing int64
			if err := tx.QueryRow(ctx, "SELECT id FROM articles WHERE url=$1", job.Args.URL).Scan(&existing); err != nil {
				return err
			}
			id = existing
		} else if err != nil {
			return err
		}
		if err := q.RecordFetchSuccess(ctx, db.RecordFetchSuccessParams{Url: job.Args.URL, ArticleID: pgtype.Int8{Int64: id, Valid: true}, Extractor: art.Extractor, BodyLength: int32(len([]rune(art.Body))), Category: art.Category, Eligible: eligible, ScopeReason: reason, PublishedAt: timestamp(art.PublishedAt)}); err != nil {
			return err
		}
		if !eligible {
			return nil
		}
		_, err = river.ClientFromContext[pgx.Tx](ctx).InsertTx(ctx, tx, DedupeArticleArgs{ArticleID: id}, nil)
		return err
	})
}
func (w *FetchArticleWorker) failure(ctx context.Context, job *river.Job[FetchArticleArgs], cause error) error {
	result, status, next := "retry", int32(0), time.Now().Add(30*time.Minute)
	var he *crawl.HTTPError
	if errors.As(cause, &he) {
		status = int32(he.Status)
		if he.Status == 429 {
			next = time.Now().Add(he.RetryAfter)
		} else if he.Status >= 400 && he.Status < 500 {
			result = "http_error"
		}
	}
	if errors.Is(cause, crawl.ErrDisallowed) {
		result = "robots_denied"
	}
	if errors.Is(cause, crawl.ErrNoArticle) {
		result = "extraction_failed"
		status = 200
	}
	if job.Attempt >= job.MaxAttempts && result == "retry" {
		result = "exhausted"
	}
	if err := db.New(w.deps.Pool).RecordFetchFailure(ctx, db.RecordFetchFailureParams{Url: job.Args.URL, Result: result, HttpStatus: status, Error: cause.Error(), NextAttemptAt: next}); err != nil {
		return err
	}
	if result != "retry" {
		return river.JobCancel(cause)
	}
	if he != nil && he.Status == 429 {
		return river.JobSnooze(time.Until(next))
	}
	return cause
}
func timestamp(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()}
}
func orEmpty(sig dedupe.Signature) []int32 {
	if sig == nil {
		return []int32{}
	}
	return sig
}

// Compatibility only: drain jobs created by the previous worker release.
type MatchHeadlineArgs struct {
	ArticleID int64 `json:"article_id"`
}

func (MatchHeadlineArgs) Kind() string { return "match_headline" }

type MatchHeadlineWorker struct {
	river.WorkerDefaults[MatchHeadlineArgs]
}

func (*MatchHeadlineWorker) Work(context.Context, *river.Job[MatchHeadlineArgs]) error { return nil }
