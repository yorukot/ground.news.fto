package httpapi

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/yorukot/ground-news-tw/internal/admin"
	"github.com/yorukot/ground-news-tw/internal/db"
)

// AdminStore is what the admin endpoints read.
type AdminStore interface {
	ListLinkDecisions(ctx context.Context, arg db.ListLinkDecisionsParams) ([]db.ListLinkDecisionsRow, error)
	SearchEventsByTitle(ctx context.Context, arg db.SearchEventsByTitleParams) ([]db.SearchEventsByTitleRow, error)
}

// AdminActions are the corrections the admin endpoints can make.
type AdminActions interface {
	MergeEvents(ctx context.Context, into, from int64) error
	MoveArticle(ctx context.Context, articleID int64, target admin.MoveTarget) (int64, error)
}

// Admin enables the /api/v1/admin endpoints. Without a token they don't exist.
type Admin struct {
	Token   string
	Store   AdminStore
	Actions AdminActions
}

type LinkDecision struct {
	ArticleID   int64     `json:"articleId"`
	Headline    string    `json:"headline"`
	OutletName  string    `json:"outletName"`
	PublishedAt time.Time `json:"publishedAt"`
	Confidence  float32   `json:"confidence"`
	EventID     int64     `json:"eventId"`
	EventTitle  string    `json:"eventTitle"`
}

type EventMatch struct {
	ID           int64     `json:"id"`
	Title        string    `json:"title"`
	UpdatedAt    time.Time `json:"updatedAt"`
	ArticleCount int64     `json:"articleCount"`
}

func (s *Server) registerAdmin(mux *http.ServeMux) {
	a := s.admin
	if a == nil || a.Token == "" {
		return
	}
	mux.Handle("GET /api/v1/admin/session", s.requireAdmin(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	mux.Handle("GET /api/v1/admin/links", s.requireAdmin(s.listLinkDecisions))
	mux.Handle("GET /api/v1/admin/events", s.requireAdmin(s.searchEvents))
	mux.Handle("POST /api/v1/admin/events/{id}/merge", s.requireAdmin(s.mergeEvents))
	mux.Handle("POST /api/v1/admin/articles/{id}/move", s.requireAdmin(s.moveArticle))
}

func (s *Server) requireAdmin(next http.HandlerFunc) http.Handler {
	// Hashing both sides makes the comparison constant-time regardless of length.
	want := sha256.Sum256([]byte(s.admin.Token))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		got := sha256.Sum256([]byte(token))
		if !ok || subtle.ConstantTimeCompare(got[:], want[:]) != 1 {
			s.writeError(w, http.StatusUnauthorized, "unauthorized", "a valid admin token is required")
			return
		}
		next(w, r)
	})
}

func (s *Server) listLinkDecisions(w http.ResponseWriter, r *http.Request) {
	maxConfidence := float32(1)
	if raw := r.URL.Query().Get("max_confidence"); raw != "" {
		v, err := strconv.ParseFloat(raw, 32)
		if err != nil || v < 0 || v > 1 {
			s.writeError(w, http.StatusBadRequest, "invalid_max_confidence", "max_confidence must be between 0 and 1")
			return
		}
		maxConfidence = float32(v)
	}
	rows, err := s.admin.Store.ListLinkDecisions(r.Context(), db.ListLinkDecisionsParams{
		MaxConfidence: pgFloat4(maxConfidence),
		MaxResults:    100,
	})
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	out := make([]LinkDecision, 0, len(rows))
	for _, row := range rows {
		out = append(out, LinkDecision{
			ArticleID: row.ID, Headline: row.Headline, OutletName: row.OutletName, PublishedAt: row.PublishedAt,
			Confidence: row.LinkConfidence.Float32, EventID: row.EventID, EventTitle: row.EventTitle,
		})
	}
	s.writeJSON(w, http.StatusOK, map[string][]LinkDecision{"links": out})
}

func (s *Server) searchEvents(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		s.writeJSON(w, http.StatusOK, map[string][]EventMatch{"events": {}})
		return
	}
	// The query is matched with ILIKE; escape its wildcards so it is literal.
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(query)
	rows, err := s.admin.Store.SearchEventsByTitle(r.Context(), db.SearchEventsByTitleParams{Query: escaped, MaxResults: 10})
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	out := make([]EventMatch, 0, len(rows))
	for _, row := range rows {
		out = append(out, EventMatch{ID: row.ID, Title: row.Title, UpdatedAt: row.UpdatedAt, ArticleCount: row.ArticleCount})
	}
	s.writeJSON(w, http.StatusOK, map[string][]EventMatch{"events": out})
}

func (s *Server) mergeEvents(w http.ResponseWriter, r *http.Request) {
	into, ok := s.pathID(w, r)
	if !ok {
		return
	}
	var body struct {
		FromEventID int64 `json:"fromEventId"`
	}
	if !s.readJSON(w, r, &body) {
		return
	}
	if err := s.admin.Actions.MergeEvents(r.Context(), into, body.FromEventID); err != nil {
		s.adminError(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]int64{"eventId": into})
}

func (s *Server) moveArticle(w http.ResponseWriter, r *http.Request) {
	articleID, ok := s.pathID(w, r)
	if !ok {
		return
	}
	var body struct {
		EventID       int64  `json:"eventId"`
		NewEventTitle string `json:"newEventTitle"`
	}
	if !s.readJSON(w, r, &body) {
		return
	}
	dest, err := s.admin.Actions.MoveArticle(r.Context(), articleID, admin.MoveTarget{EventID: body.EventID, NewEventTitle: body.NewEventTitle})
	if err != nil {
		s.adminError(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]int64{"eventId": dest})
}

func (s *Server) pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		s.writeError(w, http.StatusNotFound, "not_found", "not found")
		return 0, false
	}
	return id, true
}

func (s *Server) readJSON(w http.ResponseWriter, r *http.Request, into any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16))
	dec.DisallowUnknownFields()
	if err := dec.Decode(into); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_body", "the request body is not valid JSON for this endpoint")
		return false
	}
	return true
}

func (s *Server) adminError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, admin.ErrNotFound):
		s.writeError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, admin.ErrInvalid):
		s.writeError(w, http.StatusUnprocessableEntity, "invalid", err.Error())
	default:
		s.internalError(w, r, err)
	}
}
