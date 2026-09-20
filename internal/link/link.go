// Package link attaches an analyzed article to an existing event of any age,
// or starts a new event for it.
//
// Candidates come from two cheap lookups, shared entities and a full-text
// search of event titles and timelines, and the model makes the decision.
// Shared names only nominate a candidate; they never merge events by themselves.
package link

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yorukot/ground-news-tw/internal/ai"
	"github.com/yorukot/ground-news-tw/internal/db"
)

const (
	maxCandidates    = 5
	perSourceResults = 15
	// rrfK damps reciprocal-rank fusion so that first place in one list
	// doesn't outweigh a good showing in both.
	rrfK = 10
)

type Linker struct {
	Pool *pgxpool.Pool
	AI   ai.Client
}

// Result reports what Link did, for logging.
type Result struct {
	EventID    int64
	NewEvent   bool
	NewStep    bool
	Confidence float64
	Skipped    string
}

// Link must not run concurrently with itself: two articles about the same
// brand-new story would each find no candidate and both create an event.
// The pipeline guarantees this by giving the link queue a single worker.
func (l *Linker) Link(ctx context.Context, articleID int64) (Result, error) {
	q := db.New(l.Pool)

	art, err := q.GetArticleForLink(ctx, articleID)
	if err != nil {
		return Result{}, fmt.Errorf("load article %d: %w", articleID, err)
	}
	switch {
	case art.ReprintOfID.Valid:
		return Result{Skipped: "reprint"}, nil
	case art.EventID.Valid:
		if art.ManualLink && art.Summary != "" && !art.StepArticleID.Valid {
			err := pgx.BeginFunc(ctx, l.Pool, func(tx pgx.Tx) error {
				q := db.New(tx)
				if art.Development != "" {
					if err := q.SetArticleStep(ctx, db.SetArticleStepParams{ID: art.ID, StepArticleID: pgtype.Int8{Int64: art.ID, Valid: true}}); err != nil {
						return err
					}
				}
				if err := q.CopyArticleEntitiesToEvent(ctx, db.CopyArticleEntitiesToEventParams{ArticleID: art.ID, EventID: art.EventID.Int64}); err != nil {
					return err
				}
				return RefreshEvent(ctx, q, art.EventID.Int64, time.Now())
			})
			return Result{EventID: art.EventID.Int64, Skipped: "manual link preserved"}, err
		}
		return Result{EventID: art.EventID.Int64, Skipped: "already linked"}, nil
	case art.Summary == "":
		return Result{Skipped: "not analyzed"}, nil
	}

	entities, err := q.ListArticleEntities(ctx, articleID)
	if err != nil {
		return Result{}, fmt.Errorf("load entities: %w", err)
	}

	candidates, err := l.candidates(ctx, q, art, entities)
	if err != nil {
		return Result{}, err
	}

	entityNames := make([]string, 0, len(entities))
	for _, e := range entities {
		entityNames = append(entityNames, e.CanonicalName)
	}
	decision, err := l.AI.LinkToEvent(ctx, ai.LinkInput{
		Headline:    art.Headline,
		Summary:     art.Summary,
		Development: art.Development,
		Recaps:      art.Recaps,
		Entities:    entityNames,
		Candidates:  candidates,
	})
	if err != nil {
		return Result{}, fmt.Errorf("link decision: %w", err)
	}

	// The title call also happens before the transaction opens, so no
	// transaction is ever held open across a model call.
	var newTitle string
	if decision.EventID == 0 {
		newTitle, err = l.AI.TitleEvent(ctx, ai.TitleInput{Headline: art.Headline, Summary: art.Summary, Development: art.Development})
		if err != nil {
			return Result{}, fmt.Errorf("title event: %w", err)
		}
	}

	// A missing Chinese title is not an error: the translate job fills it in
	// later and the API serves English meanwhile.
	var newTitleZh string
	if newTitle != "" {
		names := make([]string, 0, len(entities))
		for _, e := range entities {
			names = append(names, e.CanonicalName)
		}
		if tr, err := l.AI.Translate(ctx, ai.TranslateInput{Title: newTitle, Names: names}); err != nil {
			slog.Warn("translate event title", "title", newTitle, "err", err)
		} else {
			newTitleZh = tr.Title
		}
	}

	res := Result{EventID: decision.EventID, NewEvent: decision.EventID == 0, Confidence: decision.Confidence}
	err = pgx.BeginFunc(ctx, l.Pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		if res.NewEvent {
			at := art.PublishedAt
			if at.IsZero() {
				at = art.FirstSeenAt
			}
			id, err := q.CreateEvent(ctx, db.CreateEventParams{Title: newTitle, FirstSeenAt: at, UpdatedAt: at})
			if err != nil {
				return fmt.Errorf("create event: %w", err)
			}
			res.EventID = id
			if newTitleZh != "" {
				if err := q.SetEventZh(ctx, db.SetEventZhParams{ID: id, TitleZh: newTitleZh}); err != nil {
					return fmt.Errorf("set event title: %w", err)
				}
			}
		}

		var step pgtype.Int8
		if art.Development != "" {
			step = pgtype.Int8{Int64: articleID, Valid: true}
			res.NewStep = true
			if decision.StepArticleID != 0 {
				step.Int64, res.NewStep = decision.StepArticleID, false
			}
		}
		err := q.SetArticleLink(ctx, db.SetArticleLinkParams{
			ID:             articleID,
			EventID:        pgtype.Int8{Int64: res.EventID, Valid: true},
			StepArticleID:  step,
			LinkConfidence: pgtype.Float4{Float32: float32(decision.Confidence), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("set link: %w", err)
		}
		if err := q.SetReprintsEvent(ctx, db.SetReprintsEventParams{ReprintOfID: pgtype.Int8{Int64: articleID, Valid: true}, EventID: pgtype.Int8{Int64: res.EventID, Valid: true}}); err != nil {
			return fmt.Errorf("move reprints: %w", err)
		}
		for _, e := range entities {
			if err := q.LinkEventEntity(ctx, db.LinkEventEntityParams{EventID: res.EventID, EntityID: e.ID}); err != nil {
				return fmt.Errorf("link event entity: %w", err)
			}
		}
		return RefreshEvent(ctx, q, res.EventID, time.Now())
	})
	if err != nil {
		return Result{}, err
	}
	return res, nil
}

// RefreshEvent rewrites an event's searchable timeline text and bumps its
// updated time. Anything that changes an event's steps must call it.
func RefreshEvent(ctx context.Context, q *db.Queries, eventID int64, updatedAt time.Time) error {
	steps, err := q.ListEventSteps(ctx, []int64{eventID})
	if err != nil {
		return fmt.Errorf("list steps: %w", err)
	}
	lines := make([]string, 0, len(steps))
	for _, s := range steps {
		lines = append(lines, s.Development)
	}
	err = q.TouchEvent(ctx, db.TouchEventParams{ID: eventID, UpdatedAt: updatedAt, TimelineText: strings.Join(lines, "\n")})
	if err != nil {
		return fmt.Errorf("touch event: %w", err)
	}
	return nil
}

func (l *Linker) candidates(ctx context.Context, q *db.Queries, art db.GetArticleForLinkRow, entities []db.ListArticleEntitiesRow) ([]ai.CandidateEvent, error) {
	scores := make(map[int64]float64)

	if len(entities) > 0 {
		ids := make([]int64, 0, len(entities))
		for _, e := range entities {
			ids = append(ids, e.ID)
		}
		rows, err := q.ListEventsSharingEntities(ctx, db.ListEventsSharingEntitiesParams{EntityIds: ids, MaxResults: perSourceResults})
		if err != nil {
			return nil, fmt.Errorf("entity candidates: %w", err)
		}
		for rank, row := range rows {
			scores[row.ID] += 1 / float64(rrfK+rank+1)
		}
	}

	// Recaps come first: a recap that matches an old timeline step is the
	// strongest evidence the plan identifies.
	text := strings.Join(append(append([]string{}, art.Recaps...), art.Development, art.Summary), "\n")
	if strings.TrimSpace(text) != "" {
		rows, err := q.SearchEventCandidates(ctx, db.SearchEventCandidatesParams{SearchText: text, MaxResults: perSourceResults})
		if err != nil {
			return nil, fmt.Errorf("text candidates: %w", err)
		}
		for rank, row := range rows {
			scores[row.ID] += 1 / float64(rrfK+rank+1)
		}
	}
	if len(scores) == 0 {
		return nil, nil
	}

	ids := make([]int64, 0, len(scores))
	for id := range scores {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		if scores[ids[i]] != scores[ids[j]] {
			return scores[ids[i]] > scores[ids[j]]
		}
		return ids[i] > ids[j]
	})
	if len(ids) > maxCandidates {
		ids = ids[:maxCandidates]
	}

	titles, err := q.GetEventTitles(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("event titles: %w", err)
	}
	titleOf := make(map[int64]string, len(titles))
	for _, t := range titles {
		titleOf[t.ID] = t.Title
	}
	steps, err := q.ListEventSteps(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("event steps: %w", err)
	}
	stepsOf := make(map[int64][]ai.CandidateStep)
	for _, s := range steps {
		date := s.PublishedAt.Format(time.DateOnly)
		if s.HappenedOn.Valid {
			date = s.HappenedOn.Time.Format(time.DateOnly)
		}
		stepsOf[s.EventID.Int64] = append(stepsOf[s.EventID.Int64], ai.CandidateStep{ArticleID: s.ID, HappenedOn: date, Development: s.Development})
	}

	out := make([]ai.CandidateEvent, 0, len(ids))
	for _, id := range ids {
		out = append(out, ai.CandidateEvent{ID: id, Title: titleOf[id], Steps: stepsOf[id]})
	}
	return out, nil
}
