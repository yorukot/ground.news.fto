package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"

	"github.com/yorukot/ground-news-tw/internal/ai"
	"github.com/yorukot/ground-news-tw/internal/db"
	"github.com/yorukot/ground-news-tw/internal/dedupe"
	"github.com/yorukot/ground-news-tw/internal/link"
)

var taipei = time.FixedZone("Asia/Taipei", 8*60*60)

type DedupeArticleArgs struct {
	ArticleID int64 `json:"article_id"`
}

func (DedupeArticleArgs) Kind() string { return "dedupe_article" }

type DedupeArticleWorker struct {
	river.WorkerDefaults[DedupeArticleArgs]
	deps Deps
}

func (w *DedupeArticleWorker) Work(ctx context.Context, job *river.Job[DedupeArticleArgs]) error {
	q := db.New(w.deps.Pool)
	art, err := q.GetArticleForDedupe(ctx, job.Args.ArticleID)
	if err != nil {
		return fmt.Errorf("load article: %w", err)
	}
	if art.ReprintOfID.Valid {
		return nil // an earlier attempt already marked it
	}

	var original *db.ListRecentSignaturesRow
	if len(art.Minhash) > 0 {
		recent, err := q.ListRecentSignatures(ctx, db.ListRecentSignaturesParams{Since: time.Now().Add(-reprintWindow), ExcludeID: art.ID})
		if err != nil {
			return fmt.Errorf("list signatures: %w", err)
		}
		best := dedupe.ReprintThreshold
		for i := range recent {
			if sim := dedupe.Similarity(art.Minhash, recent[i].Minhash); sim >= best {
				best, original = sim, &recent[i]
			}
		}
	}

	return pgx.BeginFunc(ctx, w.deps.Pool, func(tx pgx.Tx) error {
		if original != nil {
			// The wire story appears once; this outlet is listed under it.
			// A reprint is not summarized: its text is the original's.
			return db.New(tx).SetArticleReprint(ctx, db.SetArticleReprintParams{
				ID:          art.ID,
				ReprintOfID: pgtype.Int8{Int64: original.ID, Valid: true},
				EventID:     original.EventID,
			})
		}
		_, err := river.ClientFromContext[pgx.Tx](ctx).InsertTx(ctx, tx, SummarizeArticleArgs{ArticleID: art.ID}, nil)
		return err
	})
}

type SummarizeArticleArgs struct {
	ArticleID int64 `json:"article_id"`
}

func (SummarizeArticleArgs) Kind() string { return "summarize_article" }

func (SummarizeArticleArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueAI, MaxAttempts: 5}
}

type SummarizeArticleWorker struct {
	river.WorkerDefaults[SummarizeArticleArgs]
	deps Deps
}

func (w *SummarizeArticleWorker) Timeout(*river.Job[SummarizeArticleArgs]) time.Duration {
	return 6 * time.Minute
}

func (w *SummarizeArticleWorker) Work(ctx context.Context, job *river.Job[SummarizeArticleArgs]) error {
	q := db.New(w.deps.Pool)
	art, err := q.GetArticleForAnalysis(ctx, job.Args.ArticleID)
	if err != nil {
		return fmt.Errorf("load article: %w", err)
	}
	if art.ReprintOfID.Valid {
		return nil
	}

	// The model is called at most once per article, prompt version and
	// model: a retried job reuses the stored result and only re-enqueues.
	existing, err := q.GetSummaryVersion(ctx, art.ID)
	upToDate := err == nil && existing.Model == w.deps.AI.Model() && existing.PromptVersion == ai.PromptVersion
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("check summary: %w", err)
	}

	var analysis ai.ArticleAnalysis
	if !upToDate {
		analysis, err = w.deps.AI.SummarizeArticle(ctx, ai.ArticleInput{
			Outlet:      art.OutletName,
			Headline:    art.Headline,
			Body:        art.Body,
			PublishedOn: art.PublishedAt.In(taipei).Format(time.DateOnly),
		})
		if err != nil {
			return fmt.Errorf("summarize: %w", err)
		}
	}

	return pgx.BeginFunc(ctx, w.deps.Pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		if !upToDate {
			if err := storeAnalysis(ctx, q, art.ID, analysis, w.deps.AI.Model()); err != nil {
				return err
			}
		}
		_, err := river.ClientFromContext[pgx.Tx](ctx).InsertTx(ctx, tx, LinkArticleArgs{ArticleID: art.ID}, nil)
		return err
	})
}

func storeAnalysis(ctx context.Context, q *db.Queries, articleID int64, a ai.ArticleAnalysis, model string) error {
	var happenedOn pgtype.Date
	if a.HappenedOn != "" {
		if at, err := time.Parse(time.DateOnly, a.HappenedOn); err == nil {
			happenedOn = pgtype.Date{Time: at, Valid: true}
		}
	}
	err := q.SetArticleAnalysis(ctx, db.SetArticleAnalysisParams{
		ID: articleID,
		// A timeline line is a label, not a sentence.
		Development:       strings.TrimRight(a.Development, ". "),
		HappenedOn:        happenedOn,
		DateIsApproximate: a.DateIsApproximate && happenedOn.Valid,
	})
	if err != nil {
		return fmt.Errorf("store analysis: %w", err)
	}
	recaps := a.Recaps
	if recaps == nil {
		recaps = []string{}
	}
	err = q.UpsertSummary(ctx, db.UpsertSummaryParams{
		ArticleID:     articleID,
		Text:          a.Summary,
		Recaps:        recaps,
		Model:         model,
		PromptVersion: ai.PromptVersion,
	})
	if err != nil {
		return fmt.Errorf("store summary: %w", err)
	}
	for _, e := range a.Entities {
		id, err := upsertEntity(ctx, q, e)
		if err != nil {
			return err
		}
		if id == 0 {
			continue
		}
		if err := q.LinkArticleEntity(ctx, db.LinkArticleEntityParams{ArticleID: articleID, EntityID: id}); err != nil {
			return fmt.Errorf("link entity: %w", err)
		}
	}
	return nil
}

// upsertEntity finds an entity by any of the names the article uses for it,
// merging new aliases in, or creates it. Matching is exact and within one
// kind: a fuzzy match here would silently join unrelated people.
func upsertEntity(ctx context.Context, q *db.Queries, e ai.Entity) (int64, error) {
	name := strings.TrimSpace(e.Name)
	if name == "" {
		return 0, nil
	}
	switch e.Kind {
	case ai.KindPerson, ai.KindOrganization, ai.KindPlace:
	default:
		return 0, nil
	}

	names := []string{name}
	for _, alias := range e.Aliases {
		if alias = strings.TrimSpace(alias); alias != "" && !slices.Contains(names, alias) {
			names = append(names, alias)
		}
	}

	// Only the full name identifies an existing entity. Aliases such as
	// "王男" are shared by countless people and are stored, never matched on.
	found, err := q.FindEntity(ctx, db.FindEntityParams{Kind: string(e.Kind), Name: name})
	if errors.Is(err, pgx.ErrNoRows) {
		id, err := q.CreateEntity(ctx, db.CreateEntityParams{
			CanonicalName: name,
			Kind:          string(e.Kind),
			Aliases:       names[1:],
			SearchText:    strings.Join(names, " "),
		})
		if err != nil {
			return 0, fmt.Errorf("create entity %q: %w", name, err)
		}
		return id, nil
	}
	if err != nil {
		return 0, fmt.Errorf("find entity %q: %w", name, err)
	}

	merged := found.Aliases
	for _, n := range names {
		if n != found.CanonicalName && !slices.Contains(merged, n) {
			merged = append(merged, n)
		}
	}
	if len(merged) != len(found.Aliases) {
		err := q.SetEntityAliases(ctx, db.SetEntityAliasesParams{
			ID:         found.ID,
			Aliases:    merged,
			SearchText: strings.Join(append([]string{found.CanonicalName}, merged...), " "),
		})
		if err != nil {
			return 0, fmt.Errorf("update aliases of %q: %w", name, err)
		}
	}
	return found.ID, nil
}

type LinkArticleArgs struct {
	ArticleID int64 `json:"article_id"`
}

func (LinkArticleArgs) Kind() string { return "link_article" }

func (LinkArticleArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueLink, MaxAttempts: 5}
}

type LinkArticleWorker struct {
	river.WorkerDefaults[LinkArticleArgs]
	linker *link.Linker
}

func (w *LinkArticleWorker) Timeout(*river.Job[LinkArticleArgs]) time.Duration {
	return 8 * time.Minute
}

func (w *LinkArticleWorker) Work(ctx context.Context, job *river.Job[LinkArticleArgs]) error {
	res, err := w.linker.Link(ctx, job.Args.ArticleID)
	if err != nil {
		return err
	}
	if res.EventID != 0 {
		client := river.ClientFromContext[pgx.Tx](ctx)
		if _, err := client.Insert(ctx, SummarizeEventArgs{EventID: res.EventID}, nil); err != nil {
			return fmt.Errorf("enqueue event summary: %w", err)
		}
	}
	slog.Info("linked article", "article", job.Args.ArticleID, "event", res.EventID,
		"new_event", res.NewEvent, "new_step", res.NewStep, "confidence", res.Confidence, "skipped", res.Skipped)
	return nil
}

type SummarizeEventArgs struct {
	EventID int64 `json:"event_id"`
}

type EnqueueEventSummariesArgs struct{}

func (EnqueueEventSummariesArgs) Kind() string { return "enqueue_event_summaries" }

type EnqueueEventSummariesWorker struct {
	river.WorkerDefaults[EnqueueEventSummariesArgs]
	deps Deps
}

func (w *EnqueueEventSummariesWorker) Work(ctx context.Context, _ *river.Job[EnqueueEventSummariesArgs]) error {
	q := db.New(w.deps.Pool)
	events, err := q.ListEventsNeedingSummary(ctx, db.ListEventsNeedingSummaryParams{
		Model: w.deps.AI.Model(), PromptVersion: ai.EventSummaryPromptVersion, MaxResults: 100,
	})
	if err != nil {
		return fmt.Errorf("list events needing summaries: %w", err)
	}
	client := river.ClientFromContext[pgx.Tx](ctx)
	for _, event := range events {
		if _, err := client.Insert(ctx, SummarizeEventArgs{EventID: event}, nil); err != nil {
			return fmt.Errorf("enqueue summary for event %d: %w", event, err)
		}
	}
	return nil
}

func (SummarizeEventArgs) Kind() string { return "summarize_event" }

func (SummarizeEventArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueAI, MaxAttempts: 5}
}

type SummarizeEventWorker struct {
	river.WorkerDefaults[SummarizeEventArgs]
	deps Deps
}

func (w *SummarizeEventWorker) Timeout(*river.Job[SummarizeEventArgs]) time.Duration {
	return 4 * time.Minute
}

func (w *SummarizeEventWorker) Work(ctx context.Context, job *river.Job[SummarizeEventArgs]) error {
	q := db.New(w.deps.Pool)
	event, err := q.GetEventForSummary(ctx, job.Args.EventID)
	if err != nil {
		return fmt.Errorf("load event for summary: %w", err)
	}
	if event.SummaryModel == w.deps.AI.Model() && event.SummaryPromptVersion == ai.EventSummaryPromptVersion {
		return nil
	}
	sources, err := q.ListEventSummarySources(ctx, job.Args.EventID)
	if err != nil {
		return fmt.Errorf("load event summary sources: %w", err)
	}
	timeline := make([]string, 0, len(sources))
	summaries := make([]string, 0, len(sources))
	seenDevelopments := make(map[string]struct{})
	for _, source := range sources {
		if source.Development != "" {
			if _, exists := seenDevelopments[source.Development]; !exists {
				timeline = append(timeline, source.Development)
				seenDevelopments[source.Development] = struct{}{}
			}
		}
		if source.Summary != "" {
			summaries = append(summaries, source.Summary)
		}
	}
	if len(summaries) == 0 {
		return nil
	}
	summary, err := w.deps.AI.SummarizeEvent(ctx, ai.EventInput{
		Title: event.Title, Timeline: timeline, Summaries: summaries,
	})
	if err != nil {
		return fmt.Errorf("summarize event: %w", err)
	}
	updated, err := q.SetEventSummary(ctx, db.SetEventSummaryParams{
		ID: event.ID, Summary: summary, Model: w.deps.AI.Model(),
		PromptVersion: ai.EventSummaryPromptVersion, ExpectedUpdatedAt: event.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("store event summary: %w", err)
	}
	if updated == 0 {
		return errors.New("event changed while its summary was being generated")
	}
	return nil
}
