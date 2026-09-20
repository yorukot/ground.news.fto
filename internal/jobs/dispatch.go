package jobs

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"
	"github.com/yorukot/ground-news-tw/internal/db"
)

type DispatchAnalysisArgs struct{}

func (DispatchAnalysisArgs) Kind() string { return "dispatch_analysis" }
func (DispatchAnalysisArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{UniqueOpts: river.UniqueOpts{ByPeriod: crawlInterval}}
}

type DispatchAnalysisWorker struct {
	river.WorkerDefaults[DispatchAnalysisArgs]
	deps Deps
}

func (w *DispatchAnalysisWorker) Work(ctx context.Context, _ *river.Job[DispatchAnalysisArgs]) error {
	if !w.deps.AIEnabled {
		slog.Warn("AI dispatch paused: no model configured")
		return nil
	}
	return DispatchAnalysis(ctx, w.deps, river.ClientFromContext[pgx.Tx](ctx), time.Now())
}

// Admissions and enqueue share a transaction; the database lock serializes
// quota reservations across processes. A retry never consumes another slot.
func DispatchAnalysis(ctx context.Context, d Deps, client *river.Client[pgx.Tx], now time.Time) error {
	if !d.AIEnabled {
		return nil
	}
	return pgx.BeginFunc(ctx, d.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(72419320)"); err != nil {
			return err
		}
		day, err := time.Parse(time.DateOnly, now.In(taipei).Format(time.DateOnly))
		if err != nil {
			return err
		}
		date := pgtype.Date{Time: day, Valid: true}
		q := db.New(tx)
		count, err := q.CountDailyAdmissions(ctx, date)
		if err != nil {
			return err
		}
		for n := 0; n < 20 && count < int64(d.DailyAnalysisLimit); n++ {
			art, err := q.NextAnalysisCandidate(ctx, date)
			if errors.Is(err, pgx.ErrNoRows) {
				break
			}
			if err != nil {
				return err
			}
			if err := q.AdmitAnalysis(ctx, db.AdmitAnalysisParams{ArticleID: art.ID, OutletID: art.OutletID, AdmittedOn: date}); err != nil {
				return err
			}
			if _, err := client.InsertTx(ctx, tx, SummarizeArticleArgs{ArticleID: art.ID}, nil); err != nil {
				return err
			}
			count++
		}
		return nil
	})
}
