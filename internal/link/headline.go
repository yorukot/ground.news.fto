package link

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/yorukot/ground-news-tw/internal/db"
)

const (
	// minNameRunes keeps short, ambiguous names out of headline matching.
	// Two-character strings such as 台海 or 王男 occur in countless headlines.
	minNameRunes = 3
	// headlineWindow limits headline matches to events still in the news.
	headlineWindow = 72 * time.Hour
	// minSharedEntities is how many distinct known entities a headline must
	// share with an event. One shared name is never enough to join events.
	minSharedEntities = 2
)

// MatchHeadline attaches a headline-only article to a current event, using
// nothing but string matching: the outlet's text must not reach a model, so
// there is no model to ask. It looks for known entity names in the headline
// and picks the recent event that shares the most of them.
//
// It is deliberately conservative. It never creates an event, never adds a
// timeline step, needs two shared entities, and refuses a tie. What it does
// attach gets a modest confidence, so it shows up for review in the admin tool.
func MatchHeadline(ctx context.Context, q *db.Queries, articleID int64, now time.Time) (eventID int64, err error) {
	art, err := q.GetHeadlineArticle(ctx, articleID)
	if err != nil {
		return 0, fmt.Errorf("load article %d: %w", articleID, err)
	}
	if art.EventID.Valid {
		return art.EventID.Int64, nil
	}

	entities, err := q.ListEntityNames(ctx)
	if err != nil {
		return 0, fmt.Errorf("list entities: %w", err)
	}
	var matched []int64
	for _, e := range entities {
		if mentions(art.Headline, e.CanonicalName, e.Aliases) {
			matched = append(matched, e.ID)
		}
	}
	if len(matched) < minSharedEntities {
		return 0, nil
	}

	events, err := q.ListRecentEventsSharingEntities(ctx, db.ListRecentEventsSharingEntitiesParams{
		EntityIds:    matched,
		UpdatedSince: now.Add(-headlineWindow),
		MaxResults:   2,
	})
	if err != nil {
		return 0, fmt.Errorf("find events: %w", err)
	}
	if len(events) == 0 || events[0].Shared < minSharedEntities {
		return 0, nil
	}
	if len(events) > 1 && events[1].Shared == events[0].Shared {
		return 0, nil // two events fit equally well; don't guess
	}

	confidence := float32(0.5)
	if events[0].Shared > minSharedEntities {
		confidence = 0.7
	}
	err = q.AttachHeadlineArticle(ctx, db.AttachHeadlineArticleParams{
		ID:             articleID,
		EventID:        pgtype.Int8{Int64: events[0].ID, Valid: true},
		LinkConfidence: pgtype.Float4{Float32: confidence, Valid: true},
	})
	if err != nil {
		return 0, fmt.Errorf("attach article: %w", err)
	}
	return events[0].ID, nil
}

func mentions(headline, canonical string, aliases []string) bool {
	for _, name := range append([]string{canonical}, aliases...) {
		if utf8.RuneCountInString(name) >= minNameRunes && strings.Contains(headline, name) {
			return true
		}
	}
	return false
}
