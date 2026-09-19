package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/riverqueue/river"

	"github.com/yorukot/ground-news-tw/internal/ai"
	"github.com/yorukot/ground-news-tw/internal/db"
)

// translateBatch is how many rows one query fetches; translateMaxPerRun bounds
// one run so a large backlog is worked off over several runs, not one huge job.
const (
	translateBatch     = 20
	translateMaxPerRun = 600
	// translateWorkers is how many model calls run at once within a job.
	translateWorkers = 6
)

// TranslateMissingArgs fills in the Traditional Chinese text that the pipeline
// did not write itself: rows created before the site was bilingual, event
// titles whose translation failed, and titles typed in the admin tool.
type TranslateMissingArgs struct{}

func (TranslateMissingArgs) Kind() string { return "translate_missing" }

func (TranslateMissingArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueTranslate, MaxAttempts: 2, UniqueOpts: river.UniqueOpts{ByPeriod: crawlInterval}}
}

type TranslateMissingWorker struct {
	river.WorkerDefaults[TranslateMissingArgs]
	deps Deps
}

func (w *TranslateMissingWorker) Timeout(*river.Job[TranslateMissingArgs]) time.Duration {
	return 20 * time.Minute
}

func (w *TranslateMissingWorker) Work(ctx context.Context, _ *river.Job[TranslateMissingArgs]) error {
	q := db.New(w.deps.Pool)

	var mu sync.Mutex
	done, failed := 0, 0
	// A row that fails is skipped for the rest of this run and retried next run.
	skipped := map[string]bool{}

	// run translates the rows of one batch concurrently and reports whether any
	// succeeded. key names a row uniquely across events and articles.
	run := func(tasks []func() error, keys []string) bool {
		var wg sync.WaitGroup
		sem := make(chan struct{}, translateWorkers)
		progressed := false
		for i, task := range tasks {
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				err := task()
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					slog.Warn("translate", "row", keys[i], "err", err)
					skipped[keys[i]], failed = true, failed+1
					return
				}
				done, progressed = done+1, true
			}()
		}
		wg.Wait()
		return progressed
	}

	for done+failed < translateMaxPerRun && ctx.Err() == nil {
		events, err := q.ListEventsMissingZh(ctx, int32(translateBatch+len(skipped)))
		if err != nil {
			return fmt.Errorf("list events: %w", err)
		}
		articles, err := q.ListArticlesMissingZh(ctx, int32(translateBatch+len(skipped)))
		if err != nil {
			return fmt.Errorf("list articles: %w", err)
		}

		var tasks []func() error
		var keys []string
		for _, e := range events {
			if key := fmt.Sprintf("event %d", e.ID); !skipped[key] {
				tasks, keys = append(tasks, func() error { return w.translateEvent(ctx, q, e) }), append(keys, key)
			}
		}
		for _, a := range articles {
			if key := fmt.Sprintf("article %d", a.ID); !skipped[key] {
				tasks, keys = append(tasks, func() error { return w.translateArticle(ctx, q, a) }), append(keys, key)
			}
		}
		if !run(tasks, keys) {
			break
		}
	}
	if done > 0 || failed > 0 {
		slog.Info("translated missing text", "translated", done, "failed", failed)
	}
	return nil
}

func (w *TranslateMissingWorker) translateEvent(ctx context.Context, q *db.Queries, e db.ListEventsMissingZhRow) error {
	tr, err := w.deps.AI.Translate(ctx, ai.TranslateInput{Title: e.Title, Names: e.Names})
	if err != nil {
		return err
	}
	if tr.Title == "" {
		return errors.New("model returned an empty title")
	}
	return q.SetEventTitleZh(ctx, db.SetEventTitleZhParams{ID: e.ID, TitleZh: tr.Title})
}

func (w *TranslateMissingWorker) translateArticle(ctx context.Context, q *db.Queries, a db.ListArticlesMissingZhRow) error {
	in := ai.TranslateInput{Names: a.Names}
	if a.SummaryZh == "" {
		in.Summary = a.Summary
	}
	if a.DevelopmentZh == "" {
		in.Development = a.Development
	}
	tr, err := w.deps.AI.Translate(ctx, in)
	if err != nil {
		return err
	}
	if (in.Summary != "" && tr.Summary == "") || (in.Development != "" && tr.Development == "") {
		return errors.New("model returned an empty translation")
	}
	if tr.Summary != "" {
		if err := q.SetSummaryZh(ctx, db.SetSummaryZhParams{ArticleID: a.ID, TextZh: tr.Summary}); err != nil {
			return err
		}
	}
	if tr.Development != "" {
		if err := q.SetArticleZh(ctx, db.SetArticleZhParams{ID: a.ID, DevelopmentZh: tr.Development}); err != nil {
			return err
		}
	}
	return nil
}
