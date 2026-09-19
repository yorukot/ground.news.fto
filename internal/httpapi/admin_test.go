package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yorukot/ground-news-tw/internal/admin"
	"github.com/yorukot/ground-news-tw/internal/db"
)

type fakeAdmin struct {
	merged   [2]int64
	moved    admin.MoveTarget
	searched string
}

func (f *fakeAdmin) ListLinkDecisions(context.Context, db.ListLinkDecisionsParams) ([]db.ListLinkDecisionsRow, error) {
	return nil, nil
}

func (f *fakeAdmin) SearchEventsByTitle(_ context.Context, arg db.SearchEventsByTitleParams) ([]db.SearchEventsByTitleRow, error) {
	f.searched = arg.Query
	return nil, nil
}

func (f *fakeAdmin) MergeEvents(_ context.Context, into, from int64) error {
	if from == 404 {
		return admin.ErrNotFound
	}
	f.merged = [2]int64{into, from}
	return nil
}

func (f *fakeAdmin) MoveArticle(_ context.Context, _ int64, target admin.MoveTarget) (int64, error) {
	if target.EventID == 0 && target.NewEventTitle == "" {
		return 0, admin.ErrInvalid
	}
	f.moved = target
	return 9, nil
}

func adminServer(token string, f *fakeAdmin) http.Handler {
	var a *Admin
	if f != nil {
		a = &Admin{Token: token, Store: f, Actions: f}
	}
	return New(&fakeStore{}, a, slog.New(slog.NewTextHandler(io.Discard, nil))).Handler()
}

func do(h http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAdminRequiresTheToken(t *testing.T) {
	h := adminServer("s3cret", &fakeAdmin{})
	for name, token := range map[string]string{"none": "", "wrong": "nope", "prefix": "s3cre"} {
		if rec := do(h, "GET", "/api/v1/admin/links", token, ""); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s token: status %d, want 401", name, rec.Code)
		}
	}
	if rec := do(h, "GET", "/api/v1/admin/links", "s3cret", ""); rec.Code != http.StatusOK {
		t.Errorf("right token: status %d, want 200", rec.Code)
	}
}

func TestAdminEndpointsDoNotExistWithoutAToken(t *testing.T) {
	for name, h := range map[string]http.Handler{"no admin": adminServer("", nil), "empty token": adminServer("", &fakeAdmin{})} {
		if rec := do(h, "GET", "/api/v1/admin/links", "", ""); rec.Code != http.StatusNotFound {
			t.Errorf("%s: status %d, want 404", name, rec.Code)
		}
	}
}

func TestAdminActions(t *testing.T) {
	f := &fakeAdmin{}
	h := adminServer("t", f)

	if rec := do(h, "POST", "/api/v1/admin/events/3/merge", "t", `{"fromEventId":5}`); rec.Code != http.StatusOK || f.merged != [2]int64{3, 5} {
		t.Errorf("merge: status %d, merged %v", rec.Code, f.merged)
	}
	if rec := do(h, "POST", "/api/v1/admin/events/3/merge", "t", `{"fromEventId":404}`); rec.Code != http.StatusNotFound {
		t.Errorf("merge missing event: status %d, want 404", rec.Code)
	}
	if rec := do(h, "POST", "/api/v1/admin/articles/7/move", "t", `{"newEventTitle":"A new event"}`); rec.Code != http.StatusOK || f.moved.NewEventTitle != "A new event" {
		t.Errorf("move: status %d, moved %+v", rec.Code, f.moved)
	}
	if rec := do(h, "POST", "/api/v1/admin/articles/7/move", "t", `{}`); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("move without target: status %d, want 422", rec.Code)
	}
	if rec := do(h, "POST", "/api/v1/admin/articles/7/move", "t", `{"surprise":1}`); rec.Code != http.StatusBadRequest {
		t.Errorf("unknown field: status %d, want 400", rec.Code)
	}

	do(h, "GET", "/api/v1/admin/events?q=100%25_sure", "t", "")
	if f.searched != `100\%\_sure` {
		t.Errorf("search query = %q, want LIKE wildcards escaped", f.searched)
	}
}
