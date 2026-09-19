package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/yorukot/ground-news-tw/internal/db"
)

type fakeStore struct {
	events   []db.ListEventsRow // newest first
	articles map[int64][]db.ListEventArticlesRow
}

func (f *fakeStore) ListEvents(_ context.Context, arg db.ListEventsParams) ([]db.ListEventsRow, error) {
	var out []db.ListEventsRow
	for _, e := range f.events {
		before := e.UpdatedAt.Before(arg.CursorUpdatedAt) ||
			(e.UpdatedAt.Equal(arg.CursorUpdatedAt) && e.ID < arg.CursorID)
		if before && len(out) < int(arg.PageSize) {
			out = append(out, e)
		}
	}
	return out, nil
}

func (f *fakeStore) GetEvent(_ context.Context, id int64) (db.GetEventRow, error) {
	for _, e := range f.events {
		if e.ID == id {
			return db.GetEventRow{ID: e.ID, Title: e.Title, FirstSeenAt: e.FirstSeenAt, UpdatedAt: e.UpdatedAt}, nil
		}
	}
	return db.GetEventRow{}, pgx.ErrNoRows
}

func (f *fakeStore) ListEventArticles(_ context.Context, eventID int64) ([]db.ListEventArticlesRow, error) {
	return f.articles[eventID], nil
}

func (f *fakeStore) ListPublicOutlets(context.Context) ([]db.ListPublicOutletsRow, error) {
	return []db.ListPublicOutletsRow{{ID: 1, Slug: "cna", Name: "中央社", Domain: "cna.com.tw"}}, nil
}

func newTestServer(store Store) http.Handler {
	return New(store, nil, slog.New(slog.NewTextHandler(io.Discard, nil))).Handler()
}

func get(t *testing.T, h http.Handler, path string, into any) int {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	if into != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), into); err != nil {
			t.Fatalf("decode %s: %v\n%s", path, err, rec.Body.String())
		}
	}
	return rec.Code
}

func TestListEventsPaginates(t *testing.T) {
	base := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	store := &fakeStore{}
	for i := int64(5); i >= 1; i-- {
		store.events = append(store.events, db.ListEventsRow{ID: i, Title: "e", UpdatedAt: base.Add(time.Duration(i) * time.Hour)})
	}
	h := newTestServer(store)

	var seen []int64
	path := "/api/v1/events?limit=2"
	for range 5 {
		var page EventsPage
		if code := get(t, h, path, &page); code != http.StatusOK {
			t.Fatalf("status %d", code)
		}
		for _, e := range page.Events {
			seen = append(seen, e.ID)
		}
		if page.NextCursor == nil {
			break
		}
		path = "/api/v1/events?limit=2&cursor=" + *page.NextCursor
	}

	want := []int64{5, 4, 3, 2, 1}
	if len(seen) != len(want) {
		t.Fatalf("saw %v, want %v", seen, want)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("saw %v, want %v", seen, want)
		}
	}
}

func TestListEventsRejectsBadInput(t *testing.T) {
	h := newTestServer(&fakeStore{})
	for _, path := range []string{"/api/v1/events?limit=0", "/api/v1/events?limit=999", "/api/v1/events?cursor=%21%21"} {
		if code := get(t, h, path, nil); code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", path, code)
		}
	}
}

func TestGetEventNotFound(t *testing.T) {
	h := newTestServer(&fakeStore{})
	for _, path := range []string{"/api/v1/events/42", "/api/v1/events/abc"} {
		if code := get(t, h, path, nil); code != http.StatusNotFound {
			t.Errorf("%s: status %d, want 404", path, code)
		}
	}
}

func TestGetEventBuildsTimelineAndFoldsReprints(t *testing.T) {
	t0 := time.Date(2026, 9, 1, 2, 0, 0, 0, time.UTC)
	int8v := func(v int64) pgtype.Int8 { return pgtype.Int8{Int64: v, Valid: true} }
	row := func(id int64, outlet string, at time.Time) db.ListEventArticlesRow {
		return db.ListEventArticlesRow{ID: id, Url: "https://example.com/" + outlet, Headline: "h", PublishedAt: at, OutletID: id, OutletSlug: outlet, OutletName: outlet}
	}

	wire := row(1, "cna", t0)
	wire.Development, wire.StepArticleID = "first step", int8v(1)
	wire.HappenedOn = pgtype.Date{Time: t0, Valid: true}
	wire.Summary = "summary"

	reprint := row(2, "udn", t0.Add(time.Hour))
	reprint.ReprintOfID = int8v(1)

	follower := row(3, "ltn", t0.Add(2*time.Hour))
	follower.StepArticleID = int8v(1)

	// No explicit date: falls back to the Taipei publish date, marked approximate.
	// 17:30 UTC is already the next day in Taipei.
	undated := row(4, "tvbs", time.Date(2026, 9, 2, 17, 30, 0, 0, time.UTC))
	undated.Development, undated.StepArticleID = "second step", int8v(4)

	explainer := row(5, "cw", t0.Add(3*time.Hour))

	// A second article by an outlet already listed on the first step.
	sameOutlet := row(6, "ltn", t0.Add(4*time.Hour))
	sameOutlet.OutletID = follower.OutletID
	sameOutlet.StepArticleID = int8v(1)

	store := &fakeStore{
		events:   []db.ListEventsRow{{ID: 7, Title: "event", UpdatedAt: t0}},
		articles: map[int64][]db.ListEventArticlesRow{7: {undated, sameOutlet, explainer, follower, reprint, wire}},
	}

	var got EventDetail
	if code := get(t, newTestServer(store), "/api/v1/events/7", &got); code != http.StatusOK {
		t.Fatalf("status %d", code)
	}

	if len(got.Articles) != 5 {
		t.Fatalf("articles = %d, want 5 (reprint folded)", len(got.Articles))
	}
	var wireCard *Article
	for i := range got.Articles {
		if got.Articles[i].ID == 1 {
			wireCard = &got.Articles[i]
		}
		if got.Articles[i].ID == 2 {
			t.Error("reprint must not get its own card")
		}
	}
	if wireCard == nil || len(wireCard.Reprints) != 1 || wireCard.Reprints[0].Outlet.Slug != "udn" {
		t.Fatalf("wire card reprints = %+v", wireCard)
	}

	if len(got.Timeline) != 2 {
		t.Fatalf("timeline = %d steps, want 2", len(got.Timeline))
	}
	first, second := got.Timeline[0], got.Timeline[1]
	if first.Development != "first step" || first.HappenedOn != "2026-09-01" || first.DateIsApproximate {
		t.Errorf("first step = %+v", first)
	}
	var reporters []string
	for _, r := range first.Reports {
		reporters = append(reporters, r.Outlet.Slug)
		if r.Outlet.Slug == "udn" && r.ArticleID != 1 {
			t.Errorf("reprint report should point at the original card, got %d", r.ArticleID)
		}
	}
	if strings.Join(reporters, ",") != "cna,udn,ltn" {
		t.Errorf("first step reporters = %v, want publish order cna,udn,ltn with ltn listed once", reporters)
	}
	if second.HappenedOn != "2026-09-03" || !second.DateIsApproximate {
		t.Errorf("undated step = %+v, want Taipei date 2026-09-03 marked approximate", second)
	}
}

func TestResponsesNeverContainBody(t *testing.T) {
	rec := httptest.NewRecorder()
	store := &fakeStore{
		events:   []db.ListEventsRow{{ID: 1, Title: "t"}},
		articles: map[int64][]db.ListEventArticlesRow{1: {{ID: 1, Headline: "h"}}},
	}
	newTestServer(store).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/events/1", nil))
	if strings.Contains(rec.Body.String(), `"body"`) {
		t.Fatal("article body must never be served")
	}
}
