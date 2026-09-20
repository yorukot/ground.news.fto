package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertest"
	"github.com/riverqueue/river/rivertype"
	"github.com/yorukot/ground-news-tw/internal/ai"
	"github.com/yorukot/ground-news-tw/internal/crawl"
	"github.com/yorukot/ground-news-tw/internal/db"
	"github.com/yorukot/ground-news-tw/internal/dbtest"
)

func testClient(t *testing.T, p *pgxpool.Pool) *river.Client[pgx.Tx] {
	t.Helper()
	c, e := river.NewClient(riverpgxv5.New(p), &river.Config{})
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func testOutlet(t *testing.T, q *db.Queries, slug string, raw []byte) int64 {
	t.Helper()
	id, e := q.UpsertOutlet(context.Background(), db.UpsertOutletParams{Slug: slug, Name: slug, Domain: "example.com", Enabled: true, CrawlConfig: raw})
	if e != nil {
		t.Fatal(e)
	}
	return id
}

func TestAdmissionsSurviveConcurrentDispatchAndMidnight(t *testing.T) {
	p := dbtest.Open(t)
	q := db.New(p)
	ctx := context.Background()
	d := Deps{Pool: p, AIEnabled: true, DailyAnalysisLimit: 5}
	for _, slug := range []string{"a", "b"} {
		o := testOutlet(t, q, slug, []byte("{}"))
		for i := 0; i < 8; i++ {
			id, e := q.InsertFetchedArticle(ctx, db.InsertFetchedArticleParams{OutletID: o, Url: fmt.Sprintf("https://example.com/%s/%d", slug, i), Headline: "headline", Body: "body", ContentHash: "hash", PublishedAt: time.Now(), Minhash: []int32{}})
			if e != nil {
				t.Fatal(e)
			}
			if e = q.MarkAnalysisReady(ctx, id); e != nil {
				t.Fatal(e)
			}
		}
	}
	now := time.Date(2026, 9, 19, 15, 59, 0, 0, time.UTC)
	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for range 4 {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- DispatchAnalysis(ctx, d, testClient(t, p), now) }()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	var count, spread int
	if e := p.QueryRow(ctx, "SELECT count(*) FROM ai_admissions").Scan(&count); e != nil || count != 5 {
		t.Fatalf("count=%d err=%v", count, e)
	}
	if e := p.QueryRow(ctx, "SELECT max(n)-min(n) FROM (SELECT count(*) n FROM ai_admissions GROUP BY outlet_id) x").Scan(&spread); e != nil || spread > 1 {
		t.Fatalf("unfair spread=%d err=%v", spread, e)
	}
	if e := DispatchAnalysis(ctx, d, testClient(t, p), now.Add(2*time.Minute)); e != nil {
		t.Fatal(e)
	}
	if e := p.QueryRow(ctx, "SELECT count(*) FROM ai_admissions").Scan(&count); e != nil || count != 10 {
		t.Fatalf("midnight count=%d err=%v", count, e)
	}
	d.AIEnabled = false
	if e := DispatchAnalysis(ctx, d, testClient(t, p), now.Add(24*time.Hour)); e != nil {
		t.Fatal(e)
	}
	if e := p.QueryRow(ctx, "SELECT count(*) FROM ai_admissions").Scan(&count); e != nil || count != 10 {
		t.Fatalf("disabled count=%d err=%v", count, e)
	}
}

func TestDiscoveryStoresEverythingAndUpgradesExistingArticle(t *testing.T) {
	p := dbtest.Open(t)
	q := db.New(p)
	ctx := context.Background()
	body := strings.Repeat("這是文章的完整內容，包含記者訪問以及公共議題的背景。", 8)
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			fmt.Fprint(w, "User-agent: *\nContent-Signal: ai-input=no\nAllow: /\n")
		case "/feed":
			fmt.Fprint(w, `<rss version="2.0"><channel><title>News</title>`)
			for i := 0; i < 60; i++ {
				fmt.Fprintf(w, "<item><title>新聞 %d</title><link>%s/news/%d</link></item>", i, server.URL, i)
			}
			fmt.Fprint(w, "</channel></rss>")
		default:
			fmt.Fprintf(w, "<html><head><title>標題</title></head><body><article><h1>標題</h1><p>%s</p></article></body></html>", body)
		}
	}))
	defer server.Close()
	raw, _ := json.Marshal(crawl.Config{Feeds: []string{server.URL + "/feed"}, Selectors: crawl.Selectors{Headline: "h1", Body: "article"}})
	o := testOutlet(t, q, "test", raw)
	d := Deps{Pool: p, Fetcher: crawl.NewFetcher("TestBot", 0), AI: ai.Fake{}, AIEnabled: true, DailyAnalysisLimit: 500}
	url := server.URL + "/news/0"
	old, e := q.InsertFetchedArticle(ctx, db.InsertFetchedArticleParams{OutletID: o, Url: url, Headline: "old title", Body: "", PublishedAt: time.Time{}, ContentHash: "", Minhash: []int32{}})
	if e != nil {
		t.Fatal(e)
	}
	for range 2 {
		if _, e := DiscoverOutlet(ctx, d, o, nil); e != nil {
			t.Fatal(e)
		}
	}
	var count int
	if e := p.QueryRow(ctx, "SELECT count(*) FROM crawl_urls").Scan(&count); e != nil || count != 60 {
		t.Fatalf("count=%d err=%v", count, e)
	}
	ctx = rivertest.WorkContext(ctx, testClient(t, p))
	worker := FetchArticleWorker{deps: d}
	job := &river.Job[FetchArticleArgs]{JobRow: &rivertype.JobRow{Attempt: 1, MaxAttempts: 4}, Args: FetchArticleArgs{OutletID: o, URL: url}}
	for range 2 {
		if e := worker.Work(ctx, job); e != nil {
			t.Fatal(e)
		}
	}
	record, e := q.GetCrawlURL(ctx, url)
	if e != nil || record.ArticleID.Int64 != old || record.BodyLength < 120 || record.Result != "success" {
		t.Fatalf("upgrade=%+v err=%v", record, e)
	}
	art, e := q.GetArticleForAnalysis(ctx, old)
	if e != nil || art.Body != body || !art.PublishedAt.IsZero() {
		t.Fatalf("article=%+v err=%v", art, e)
	}
}

func TestDiscoveryMergesMetadata(t *testing.T) {
	p := dbtest.Open(t)
	q := db.New(p)
	ctx := context.Background()
	o := testOutlet(t, q, "a", []byte("{}"))
	arg := db.DiscoverURLParams{Url: "https://example.com/a", OutletID: o, Eligible: true, ScopeReason: "unknown section"}
	if e := q.DiscoverURL(ctx, arg); e != nil {
		t.Fatal(e)
	}
	arg.Headline = "新聞"
	arg.Category = "政治"
	arg.PublishedAt = timestamp(time.Now())
	if e := q.DiscoverURL(ctx, arg); e != nil {
		t.Fatal(e)
	}
	row, e := q.GetCrawlURL(ctx, arg.Url)
	if e != nil || row.Headline != "新聞" || !row.PublishedAt.Valid || row.Category != "政治" {
		t.Fatalf("row=%+v err=%v", row, e)
	}
}

func TestReprintCandidatesOnlyPointBackward(t *testing.T) {
	p := dbtest.Open(t)
	q := db.New(p)
	ctx := context.Background()
	o := testOutlet(t, q, "wire", []byte("{}"))
	ids := make([]int64, 3)
	for i := range ids {
		var err error
		ids[i], err = q.InsertFetchedArticle(ctx, db.InsertFetchedArticleParams{OutletID: o, Url: fmt.Sprintf("https://example.com/wire/%d", i), Headline: "wire", Body: "body", PublishedAt: time.Now(), Minhash: []int32{1, 2, 3}})
		if err != nil {
			t.Fatal(err)
		}
	}
	for i, id := range ids {
		rows, err := q.ListRecentSignatures(ctx, db.ListRecentSignaturesParams{Since: time.Now().Add(-time.Hour), ExcludeID: id})
		if err != nil || len(rows) != i {
			t.Fatalf("article %d: candidates=%v err=%v", id, rows, err)
		}
		for _, row := range rows {
			if row.ID >= id {
				t.Fatalf("forward reference could create a reprint cycle: %d -> %d", id, row.ID)
			}
		}
	}
}
