// Package httpapi serves the JSON API described in api/openapi.yaml.
package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/yorukot/ground-news-tw/internal/db"
)

const (
	defaultPageSize = 20
	maxPageSize     = 50
)

// Store is the slice of the database the public API reads.
type Store interface {
	ListEvents(ctx context.Context, arg db.ListEventsParams) ([]db.ListEventsRow, error)
	GetEvent(ctx context.Context, id int64) (db.GetEventRow, error)
	ListEventArticles(ctx context.Context, eventID int64) ([]db.ListEventArticlesRow, error)
	ListPublicOutlets(ctx context.Context) ([]db.ListPublicOutletsRow, error)
}

type Server struct {
	store Store
	admin *Admin
	log   *slog.Logger
}

// New builds the API. admin may be nil, which leaves the admin endpoints out.
func New(store Store, admin *Admin, log *slog.Logger) *Server {
	return &Server{store: store, admin: admin, log: log}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /api/v1/events", s.listEvents)
	mux.HandleFunc("GET /api/v1/events/{id}", s.getEvent)
	mux.HandleFunc("GET /api/v1/outlets", s.listOutlets)
	s.registerAdmin(mux)
	return s.recoverer(s.logger(mux))
}

// lang is the language of the site's own text (titles, summaries, timeline
// lines). Original headlines and outlet names are never translated.
type lang string

const (
	langEN lang = "en"
	langZH lang = "zh-TW"
)

// parseLang reads the optional ?lang= parameter; English is the default.
func parseLang(r *http.Request) (lang, bool) {
	switch raw := r.URL.Query().Get("lang"); raw {
	case "", "en":
		return langEN, true
	case "zh-TW":
		return langZH, true
	default:
		return "", false
	}
}

// text picks the text for l, falling back to English while a translation is
// still missing so that a page never shows a hole.
func (l lang) text(en, zh string) string {
	if l == langZH && zh != "" {
		return zh
	}
	return en
}

func (s *Server) listEvents(w http.ResponseWriter, r *http.Request) {
	l, ok := parseLang(r)
	if !ok {
		s.writeError(w, http.StatusBadRequest, "invalid_lang", "lang must be en or zh-TW")
		return
	}
	params := db.ListEventsParams{
		CursorUpdatedAt: time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC),
		CursorID:        math.MaxInt64,
		PageSize:        defaultPageSize,
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > maxPageSize {
			s.writeError(w, http.StatusBadRequest, "invalid_limit", fmt.Sprintf("limit must be between 1 and %d", maxPageSize))
			return
		}
		params.PageSize = int32(n)
	}
	if raw := r.URL.Query().Get("cursor"); raw != "" {
		at, id, err := decodeCursor(raw)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid_cursor", "cursor is not valid")
			return
		}
		params.CursorUpdatedAt, params.CursorID = at, id
	}

	// Fetch one extra row to learn whether another page exists.
	pageSize := int(params.PageSize)
	params.PageSize++
	rows, err := s.store.ListEvents(r.Context(), params)
	if err != nil {
		s.internalError(w, r, err)
		return
	}

	page := EventsPage{Events: make([]EventSummary, 0, len(rows))}
	if len(rows) > pageSize {
		rows = rows[:pageSize]
		last := rows[len(rows)-1]
		cursor := encodeCursor(last.UpdatedAt, last.ID)
		page.NextCursor = &cursor
	}
	for _, row := range rows {
		page.Events = append(page.Events, EventSummary{
			ID:                row.ID,
			Title:             l.text(row.Title, row.TitleZh),
			FirstSeenAt:       row.FirstSeenAt,
			UpdatedAt:         row.UpdatedAt,
			ArticleCount:      row.ArticleCount,
			OutletCount:       row.OutletCount,
			LatestDevelopment: l.text(row.LatestDevelopment, row.LatestDevelopmentZh),
		})
	}
	s.writeJSON(w, http.StatusOK, page)
}

func (s *Server) getEvent(w http.ResponseWriter, r *http.Request) {
	l, ok := parseLang(r)
	if !ok {
		s.writeError(w, http.StatusBadRequest, "invalid_lang", "lang must be en or zh-TW")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		s.writeError(w, http.StatusNotFound, "not_found", "event not found")
		return
	}
	event, err := s.store.GetEvent(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeError(w, http.StatusNotFound, "not_found", "event not found")
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	rows, err := s.store.ListEventArticles(r.Context(), id)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	articles, timeline := buildEventBody(rows, l)
	s.writeJSON(w, http.StatusOK, EventDetail{
		ID:           event.ID,
		Title:        l.text(event.Title, event.TitleZh),
		FirstSeenAt:  event.FirstSeenAt,
		UpdatedAt:    event.UpdatedAt,
		ArticleCount: event.ArticleCount,
		OutletCount:  event.OutletCount,
		Timeline:     timeline,
		Articles:     articles,
	})
}

func (s *Server) listOutlets(w http.ResponseWriter, r *http.Request) {
	rows, err := s.store.ListPublicOutlets(r.Context())
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	outlets := make([]Outlet, 0, len(rows))
	for _, row := range rows {
		outlets = append(outlets, Outlet{ID: row.ID, Slug: row.Slug, Name: row.Name, Domain: row.Domain})
	}
	s.writeJSON(w, http.StatusOK, map[string][]Outlet{"outlets": outlets})
}

// The cursor is opaque to clients: "<updated_at unix micros>:<id>", base64url.
// Microseconds match Postgres timestamptz precision.
func encodeCursor(at time.Time, id int64) string {
	raw := strconv.FormatInt(at.UnixMicro(), 10) + ":" + strconv.FormatInt(id, 10)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(s string) (time.Time, int64, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return time.Time{}, 0, err
	}
	micros, idPart, ok := strings.Cut(string(raw), ":")
	if !ok {
		return time.Time{}, 0, errors.New("malformed cursor")
	}
	us, err := strconv.ParseInt(micros, 10, 64)
	if err != nil {
		return time.Time{}, 0, err
	}
	id, err := strconv.ParseInt(idPart, 10, 64)
	if err != nil {
		return time.Time{}, 0, err
	}
	return time.UnixMicro(us).UTC(), id, nil
}

func pgFloat4(v float32) pgtype.Float4 { return pgtype.Float4{Float32: v, Valid: true} }

func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		s.log.Error("write response", "error", err)
	}
}

func (s *Server) writeError(w http.ResponseWriter, status int, code, message string) {
	s.writeJSON(w, status, errorBody{Error: errorDetail{Code: code, Message: message}})
}

func (s *Server) internalError(w http.ResponseWriter, r *http.Request, err error) {
	s.log.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
	s.writeError(w, http.StatusInternalServerError, "internal", "internal server error")
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (s *Server) logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		s.log.Info("request", "method", r.Method, "path", r.URL.Path, "status", rec.status, "duration", time.Since(start))
	})
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				s.log.Error("panic", "path", r.URL.Path, "panic", v)
				s.writeError(w, http.StatusInternalServerError, "internal", "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
