// Package seed registers the outlet list and, for development, a few
// fictional sample events.
package seed

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yorukot/ground-news-tw/internal/db"
)

// Run upserts the outlets and, when withSamples is set, inserts the sample
// events once. It is safe to run repeatedly.
func Run(ctx context.Context, pool *pgxpool.Pool, withSamples bool) error {
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)

		outletIDs := make(map[string]int64, len(Outlets))
		for _, o := range Outlets {
			crawlConfig := []byte("{}")
			if o.Crawl != nil {
				var err error
				if crawlConfig, err = json.Marshal(o.Crawl); err != nil {
					return fmt.Errorf("outlet %s crawl config: %w", o.Slug, err)
				}
			}
			id, err := q.UpsertOutlet(ctx, db.UpsertOutletParams{
				Slug:         o.Slug,
				Name:         o.Name,
				Domain:       o.Domain,
				IsAggregator: o.IsAggregator,
				Enabled:      o.Crawl != nil,
				CrawlConfig:  crawlConfig,
				Coverage:     cmp.Or(o.Coverage, CoverageFull),
			})
			if err != nil {
				return fmt.Errorf("upsert outlet %s: %w", o.Slug, err)
			}
			outletIDs[o.Slug] = id
		}
		slog.Info("outlets registered", "count", len(Outlets))

		if !withSamples {
			return nil
		}

		var existing int
		err := tx.QueryRow(ctx, "SELECT count(*) FROM articles WHERE url LIKE $1", sampleURLPrefix+"%").Scan(&existing)
		if err != nil {
			return fmt.Errorf("check existing samples: %w", err)
		}
		if existing > 0 {
			slog.Info("sample events already present; skipping", "articles", existing)
			return nil
		}
		return insertSamples(ctx, q, outletIDs, time.Now())
	})
}

func insertSamples(ctx context.Context, q *db.Queries, outletIDs map[string]int64, now time.Time) error {
	for _, ev := range samples {
		first, last := now, time.Time{}
		for _, a := range ev.Articles {
			at := now.Add(-a.Ago)
			if at.Before(first) {
				first = at
			}
			if at.After(last) {
				last = at
			}
		}
		eventID, err := q.CreateEvent(ctx, db.CreateEventParams{Title: ev.Title, FirstSeenAt: first, UpdatedAt: last})
		if err != nil {
			return fmt.Errorf("create event %q: %w", ev.Title, err)
		}

		// Originals and first reporters come before the articles that refer
		// to them in the sample data, so one pass is enough.
		ids := make(map[string]int64, len(ev.Articles))
		var developments []string
		for _, a := range ev.Articles {
			if a.Development != "" {
				developments = append(developments, a.Development)
			}
			outletID, ok := outletIDs[a.Outlet]
			if !ok {
				return fmt.Errorf("sample %s: unknown outlet %q", a.Key, a.Outlet)
			}
			publishedAt := now.Add(-a.Ago)
			params := db.CreateArticleParams{
				OutletID:          outletID,
				EventID:           pgtype.Int8{Int64: eventID, Valid: true},
				Url:               sampleURLPrefix + a.Key,
				Headline:          a.Headline,
				Body:              a.Summary, // samples have no real body
				PublishedAt:       publishedAt,
				FirstSeenAt:       publishedAt,
				ContentHash:       hash(a.Key),
				Development:       a.Development,
				DateIsApproximate: a.Approximate,
			}
			if a.HappenedAgo != nil {
				params.HappenedOn = pgtype.Date{Time: now.Add(-*a.HappenedAgo), Valid: true}
			}
			if a.ReprintOf != "" {
				orig, ok := ids[a.ReprintOf]
				if !ok {
					return fmt.Errorf("sample %s: reprint of unknown %q", a.Key, a.ReprintOf)
				}
				params.ReprintOfID = pgtype.Int8{Int64: orig, Valid: true}
			}
			id, err := q.CreateArticle(ctx, params)
			if err != nil {
				return fmt.Errorf("create article %s: %w", a.Key, err)
			}
			ids[a.Key] = id

			stepID := int64(0)
			switch {
			case a.Development != "":
				stepID = id
			case a.StepOf != "":
				if stepID, ok = ids[a.StepOf]; !ok {
					return fmt.Errorf("sample %s: step of unknown %q", a.Key, a.StepOf)
				}
			}
			if stepID != 0 {
				if err := q.SetArticleStep(ctx, db.SetArticleStepParams{ID: id, StepArticleID: pgtype.Int8{Int64: stepID, Valid: true}}); err != nil {
					return fmt.Errorf("set step for %s: %w", a.Key, err)
				}
			}

			if a.Summary != "" {
				err := q.UpsertSummary(ctx, db.UpsertSummaryParams{
					ArticleID:     id,
					Text:          a.Summary,
					Recaps:        []string{},
					Model:         "sample",
					PromptVersion: "sample",
				})
				if err != nil {
					return fmt.Errorf("summary for %s: %w", a.Key, err)
				}
			}
		}
		err = q.SetEventTimelineText(ctx, db.SetEventTimelineTextParams{
			ID:           eventID,
			TimelineText: strings.Join(developments, "\n"),
		})
		if err != nil {
			return fmt.Errorf("timeline text for %q: %w", ev.Title, err)
		}
		slog.Info("sample event inserted", "title", ev.Title, "articles", len(ev.Articles))
	}
	return nil
}

func hash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
