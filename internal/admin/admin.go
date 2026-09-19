// Package admin holds the corrections an administrator can make to event
// linking: merging two events, and moving one article to another event. The
// model will make mistakes; these are how they get fixed.
package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yorukot/ground-news-tw/internal/db"
	"github.com/yorukot/ground-news-tw/internal/link"
)

var (
	ErrNotFound = errors.New("not found")
	ErrInvalid  = errors.New("invalid request")
)

type Service struct {
	Pool *pgxpool.Pool
}

// MergeEvents moves everything in event `from` into event `into` and deletes
// `from`. Steps keep their step articles, so both timelines interleave by date.
func (s *Service) MergeEvents(ctx context.Context, into, from int64) error {
	if into == from {
		return fmt.Errorf("%w: an event cannot be merged into itself", ErrInvalid)
	}
	return pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		// Lock in id order so two opposite merges can't deadlock.
		first, second := min(into, from), max(into, from)
		locked := make(map[int64]db.LockEventRow, 2)
		for _, id := range []int64{first, second} {
			row, err := q.LockEvent(ctx, id)
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("%w: event %d", ErrNotFound, id)
			}
			if err != nil {
				return err
			}
			locked[id] = row
		}

		if err := q.MoveEventArticles(ctx, db.MoveEventArticlesParams{ToEventID: pgtype.Int8{Int64: into, Valid: true}, FromEventID: pgtype.Int8{Int64: from, Valid: true}}); err != nil {
			return fmt.Errorf("move articles: %w", err)
		}
		if err := q.CopyEventEntities(ctx, db.CopyEventEntitiesParams{ToEventID: into, FromEventID: from}); err != nil {
			return fmt.Errorf("copy entities: %w", err)
		}
		if err := q.SetEventFirstSeen(ctx, db.SetEventFirstSeenParams{ID: into, FirstSeenAt: locked[from].FirstSeenAt}); err != nil {
			return err
		}
		if err := q.DeleteEvent(ctx, from); err != nil {
			return fmt.Errorf("delete merged event: %w", err)
		}
		// A correction is not news: the event keeps its place on the homepage.
		return link.RefreshEvent(ctx, q, into, time.Time{})
	})
}

// MoveTarget says where an article goes: an existing event, or a new one.
type MoveTarget struct {
	EventID       int64
	NewEventTitle string
}

// MoveArticle moves one article, and its reprints, to another event. It
// returns the id of the event the article ends up in.
func (s *Service) MoveArticle(ctx context.Context, articleID int64, target MoveTarget) (int64, error) {
	title := strings.TrimSpace(target.NewEventTitle)
	if (target.EventID == 0) == (title == "") {
		return 0, fmt.Errorf("%w: give either an event id or a title for a new event", ErrInvalid)
	}

	var destination int64
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		art, err := q.GetArticleForMove(ctx, articleID)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: article %d", ErrNotFound, articleID)
		}
		if err != nil {
			return err
		}
		if art.ReprintOfID.Valid {
			return fmt.Errorf("%w: article %d is a reprint; move its original (%d) instead", ErrInvalid, articleID, art.ReprintOfID.Int64)
		}

		destination = target.EventID
		if destination != 0 {
			if _, err := q.LockEvent(ctx, destination); errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("%w: event %d", ErrNotFound, destination)
			} else if err != nil {
				return err
			}
			if art.EventID.Valid && art.EventID.Int64 == destination {
				return fmt.Errorf("%w: the article is already in event %d", ErrInvalid, destination)
			}
		} else {
			now := time.Now()
			if destination, err = q.CreateEvent(ctx, db.CreateEventParams{Title: title, FirstSeenAt: now, UpdatedAt: now}); err != nil {
				return fmt.Errorf("create event: %w", err)
			}
		}

		// If the article is a timeline step that other articles attach to,
		// the earliest of them takes the step over in the old event.
		if art.EventID.Valid && art.StepArticleID.Valid && art.StepArticleID.Int64 == art.ID {
			followers, err := q.ListStepFollowers(ctx, db.ListStepFollowersParams{StepArticleID: pgtype.Int8{Int64: art.ID, Valid: true}, EventID: art.EventID})
			if err != nil {
				return err
			}
			if len(followers) > 0 {
				err := q.RepointStep(ctx, db.RepointStepParams{
					NewStepID: pgtype.Int8{Int64: followers[0].ID, Valid: true},
					OldStepID: pgtype.Int8{Int64: art.ID, Valid: true},
					EventID:   art.EventID,
				})
				if err != nil {
					return fmt.Errorf("hand the step over: %w", err)
				}
			}
		}

		// In its new event the article stands as its own step, if it reports one.
		var step pgtype.Int8
		if art.Development != "" {
			step = pgtype.Int8{Int64: art.ID, Valid: true}
		}
		dest := pgtype.Int8{Int64: destination, Valid: true}
		if err := q.MoveArticle(ctx, db.MoveArticleParams{ID: art.ID, EventID: dest, StepArticleID: step}); err != nil {
			return fmt.Errorf("move article: %w", err)
		}
		if err := q.SetReprintsEvent(ctx, db.SetReprintsEventParams{ReprintOfID: pgtype.Int8{Int64: art.ID, Valid: true}, EventID: dest}); err != nil {
			return fmt.Errorf("move reprints: %w", err)
		}
		if err := q.CopyArticleEntitiesToEvent(ctx, db.CopyArticleEntitiesToEventParams{EventID: destination, ArticleID: art.ID}); err != nil {
			return err
		}
		if err := link.RefreshEvent(ctx, q, destination, time.Time{}); err != nil {
			return err
		}
		if art.EventID.Valid {
			if err := link.RefreshEvent(ctx, q, art.EventID.Int64, time.Time{}); err != nil {
				return err
			}
			// An event left with no articles has nothing to show.
			if err := q.DeleteEventIfEmpty(ctx, art.EventID.Int64); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return destination, nil
}
