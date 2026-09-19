package admin_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/yorukot/ground-news-tw/internal/admin"
	"github.com/yorukot/ground-news-tw/internal/db"
	"github.com/yorukot/ground-news-tw/internal/dbtest"
)

type world struct {
	t      *testing.T
	q      *db.Queries
	svc    *admin.Service
	outlet int64
	n      int
}

func newWorld(t *testing.T) *world {
	pool := dbtest.Open(t)
	q := db.New(pool)
	outlet, err := q.UpsertOutlet(context.Background(), db.UpsertOutletParams{Slug: "test", Name: "Test", Domain: "example.com", CrawlConfig: []byte("{}"), Coverage: "full"})
	if err != nil {
		t.Fatal(err)
	}
	return &world{t: t, q: q, svc: &admin.Service{Pool: pool}, outlet: outlet}
}

func (w *world) event(title string) int64 {
	w.t.Helper()
	id, err := w.q.CreateEvent(context.Background(), db.CreateEventParams{Title: title, FirstSeenAt: time.Now(), UpdatedAt: time.Now()})
	if err != nil {
		w.t.Fatal(err)
	}
	return id
}

// article adds an article to an event; stepOf 0 makes it its own step.
func (w *world) article(event int64, development string, stepOf int64) int64 {
	w.t.Helper()
	ctx := context.Background()
	w.n++
	id, err := w.q.InsertFetchedArticle(ctx, db.InsertFetchedArticleParams{
		OutletID: w.outlet, Url: fmt.Sprintf("https://example.com/%d", w.n), Headline: "h", Body: "b",
		PublishedAt: time.Now().Add(time.Duration(w.n) * time.Minute), ContentHash: fmt.Sprint(w.n), Minhash: []int32{},
	})
	if err != nil {
		w.t.Fatal(err)
	}
	if err := w.q.SetArticleAnalysis(ctx, db.SetArticleAnalysisParams{ID: id, Development: development}); err != nil {
		w.t.Fatal(err)
	}
	step := id
	if stepOf != 0 {
		step = stepOf
	}
	err = w.q.SetArticleLink(ctx, db.SetArticleLinkParams{
		ID: id, EventID: pgtype.Int8{Int64: event, Valid: true},
		StepArticleID: pgtype.Int8{Int64: step, Valid: true}, LinkConfidence: pgtype.Float4{Float32: 0.4, Valid: true},
	})
	if err != nil {
		w.t.Fatal(err)
	}
	return id
}

func (w *world) steps(event int64) []int64 {
	w.t.Helper()
	rows, err := w.q.ListEventSteps(context.Background(), []int64{event})
	if err != nil {
		w.t.Fatal(err)
	}
	ids := make([]int64, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	return ids
}

func TestMergeEvents(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	a, b := w.event("Arrest of Wang"), w.event("Wang trial")
	arrest := w.article(a, "Police arrest Wang", 0)
	verdict := w.article(b, "Court convicts Wang", 0)

	if err := w.svc.MergeEvents(ctx, a, b); err != nil {
		t.Fatal(err)
	}
	if got := w.steps(a); len(got) != 2 || got[0] != arrest || got[1] != verdict {
		t.Fatalf("merged timeline = %v, want [%d %d]", got, arrest, verdict)
	}
	if _, err := w.q.GetEvent(ctx, b); err == nil {
		t.Fatal("the merged-away event should be gone")
	}
	found, err := w.q.SearchEventCandidates(ctx, db.SearchEventCandidatesParams{SearchText: "convicts", MaxResults: 5})
	if err != nil || len(found) != 1 || found[0].ID != a {
		t.Fatalf("search after merge = %+v, %v", found, err)
	}

	if err := w.svc.MergeEvents(ctx, a, a); !errors.Is(err, admin.ErrInvalid) {
		t.Errorf("self-merge err = %v, want ErrInvalid", err)
	}
	if err := w.svc.MergeEvents(ctx, a, 9999); !errors.Is(err, admin.ErrNotFound) {
		t.Errorf("missing event err = %v, want ErrNotFound", err)
	}
}

func TestMoveArticleHandsItsStepOver(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	from, to := w.event("Mixed-up event"), w.event("Right event")
	owner := w.article(from, "The council passes the budget", 0)
	follower := w.article(from, "Council approves the budget bill", owner)

	dest, err := w.svc.MoveArticle(ctx, owner, admin.MoveTarget{EventID: to})
	if err != nil || dest != to {
		t.Fatalf("move = %d, %v", dest, err)
	}
	// The follower now carries the step in the old event; the moved article
	// is its own step in the new one.
	if got := w.steps(from); len(got) != 1 || got[0] != follower {
		t.Fatalf("old event steps = %v, want [%d]", got, follower)
	}
	if got := w.steps(to); len(got) != 1 || got[0] != owner {
		t.Fatalf("new event steps = %v, want [%d]", got, owner)
	}
}

func TestMoveArticleToNewEventRemovesAnEmptiedOne(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	old := w.event("Wrong event")
	only := w.article(old, "A separate incident", 0)

	dest, err := w.svc.MoveArticle(ctx, only, admin.MoveTarget{NewEventTitle: "A separate incident in Sample City"})
	if err != nil || dest == old {
		t.Fatalf("move = %d, %v", dest, err)
	}
	if _, err := w.q.GetEvent(ctx, old); err == nil {
		t.Fatal("an event left with no articles should be deleted")
	}

	if _, err := w.svc.MoveArticle(ctx, only, admin.MoveTarget{}); !errors.Is(err, admin.ErrInvalid) {
		t.Errorf("no target err = %v, want ErrInvalid", err)
	}
	if _, err := w.svc.MoveArticle(ctx, only, admin.MoveTarget{EventID: dest}); !errors.Is(err, admin.ErrInvalid) {
		t.Errorf("same event err = %v, want ErrInvalid", err)
	}
}
