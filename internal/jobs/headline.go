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
	"github.com/yorukot/ground-news-tw/internal/link"
)

const (
	coverageHeadline = "headline"
	// An event for a headline usually appears once a full-coverage outlet
	// reports the same story, so matching is retried for a while.
	headlineRetryEvery = 30 * time.Minute
	headlineGiveUp     = 24 * time.Hour
)

// storeHeadlines records what a headline-only outlet's feed lists: headline,
// link and time. The article pages are never fetched.
func storeHeadlines(ctx context.Context, d Deps, outletID int64, found []crawl.Found) (int, error) {
	stored := 0
	for _, f := range found {
		if stored >= d.MaxNewPerCrawl {
			break
		}
		if f.Title == "" || f.PublishedAt.IsZero() || time.Since(f.PublishedAt) > maxArticleAge {
			continue
		}
		sum := sha256.Sum256([]byte(f.Title))
		err := pgx.BeginFunc(ctx, d.Pool, func(tx pgx.Tx) error {
			id, err := db.New(tx).InsertHeadlineArticle(ctx, db.InsertHeadlineArticleParams{
				OutletID:    outletID,
				Url:         f.URL,
				Headline:    f.Title,
				ImageUrl:    f.ImageURL,
				PublishedAt: f.PublishedAt,
				ContentHash: hex.EncodeToString(sum[:]),
			})
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			if err != nil {
				return err
			}
			stored++
			_, err = river.ClientFromContext[pgx.Tx](ctx).InsertTx(ctx, tx, MatchHeadlineArgs{ArticleID: id}, nil)
			return err
		})
		if err != nil {
			return stored, fmt.Errorf("store headline %s: %w", f.URL, err)
		}
	}
	return stored, nil
}

type MatchHeadlineArgs struct {
	ArticleID int64 `json:"article_id"`
}

func (MatchHeadlineArgs) Kind() string { return "match_headline" }

func (MatchHeadlineArgs) InsertOpts() river.InsertOpts {
	// Snoozes don't count as attempts; this only bounds real failures.
	return river.InsertOpts{MaxAttempts: 5}
}

type MatchHeadlineWorker struct {
	river.WorkerDefaults[MatchHeadlineArgs]
	deps Deps
}

func (w *MatchHeadlineWorker) Work(ctx context.Context, job *river.Job[MatchHeadlineArgs]) error {
	q := db.New(w.deps.Pool)
	eventID, err := link.MatchHeadline(ctx, q, job.Args.ArticleID, time.Now())
	if err != nil {
		return err
	}
	if eventID != 0 {
		slog.Info("matched headline", "article", job.Args.ArticleID, "event", eventID)
		return nil
	}
	art, err := q.GetHeadlineArticle(ctx, job.Args.ArticleID)
	if err != nil {
		return err
	}
	if time.Since(art.FirstSeenAt) < headlineGiveUp {
		return river.JobSnooze(headlineRetryEvery)
	}
	return nil // no event covers it; the article stays unlisted
}
