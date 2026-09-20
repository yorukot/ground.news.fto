package crawl

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestPaginationResumesAndRejectsExternalNext(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			fmt.Fprint(w, "User-agent: *\nAllow: /\n")
		case "/1":
			fmt.Fprint(w, `<a class="article" href="/news/1?utm_source=x">one</a><a class="next" href="/2">next</a>`)
		case "/2":
			fmt.Fprint(w, `<a class="article" href="/news/2">two</a><a class="next" href="https://other.invalid/list">next</a>`)
		}
	}))
	defer s.Close()
	f := NewFetcher("test", 0)
	src := Source{URL: s.URL + "/1", Kind: "list", List: ListSource{Links: "a.article", Next: "a.next"}}
	items, p, e := f.DiscoverSource(context.Background(), Config{}, src, Progress{}, 1)
	if e != nil || len(items) != 1 || p.Done {
		t.Fatalf("first=%+v %+v %v", items, p, e)
	}
	items, p, e = f.DiscoverSource(context.Background(), Config{}, src, p, 1)
	if e != nil || len(items) != 1 || !p.Done || !strings.HasSuffix(items[0].URL, "/news/2") {
		t.Fatalf("second=%+v %+v %v", items, p, e)
	}
}

func TestRedirectChecksDestinationRobots(t *testing.T) {
	hit := false
	dest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			fmt.Fprint(w, "User-agent: *\nDisallow: /blocked\n")
			return
		}
		hit = true
	}))
	defer dest.Close()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			return
		}
		http.Redirect(w, r, dest.URL+"/blocked", 302)
	}))
	defer s.Close()
	_, _, e := NewFetcher("test", 0).Get(context.Background(), s.URL+"/start")
	if !errors.Is(e, ErrDisallowed) || hit {
		t.Fatalf("err=%v hit=%v", e, hit)
	}
}

func TestShortSelectorFallsBackToJSONLD(t *testing.T) {
	body := strings.Repeat("完整新聞本文包含重要的公共議題以及所有相關背景。", 8)
	u, _ := url.Parse("https://example.com/article")
	page := `<html><head><title>標題</title><script type="application/ld+json">{"@type":"NewsArticle","articleBody":"` + body + `"}</script></head><body><div class="wrong">太短</div></body></html>`
	a, e := Extract([]byte(page), u, Selectors{Body: ".wrong"})
	if e != nil || a.Body != body || a.Extractor != "json-ld" {
		t.Fatalf("article=%+v err=%v", a, e)
	}
}

func TestScopeUsesSectionsNotHeadlines(t *testing.T) {
	for category, want := range map[string]bool{"政治": true, "sports": false, "政治,娛樂": true, "unknown": true, "": true} {
		got, _ := InScope(category)
		if got != want {
			t.Errorf("%q=%v", category, got)
		}
	}
}

func TestLTNListAcceptsArrayAndIndexedObject(t *testing.T) {
	u, _ := url.Parse("https://news.ltn.com.tw/ajax/breakingnews/politics/1")
	for _, data := range []string{`[{"url":"https://news.ltn.com.tw/news/politics/1","title":"標題"}]`, `{"20":{"url":"https://news.ltn.com.tw/news/politics/1","title":"標題"}}`} {
		items, next, err := parseLTNList([]byte("\xef\xbb\xbf"+`{"code":200,"data":`+data+`}`), u)
		if err != nil || len(items) != 1 || items[0].Title != "標題" || len(next) != 1 || !strings.HasSuffix(next[0], "/2") {
			t.Fatalf("items=%+v next=%v err=%v", items, next, err)
		}
	}
}

func TestSourceFailureRetainsSuccessfulPages(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			return
		case "/1":
			fmt.Fprint(w, `<a class="article" href="/news/1">one</a><a class="next" href="/2">next</a>`)
		default:
			w.WriteHeader(503)
		}
	}))
	defer s.Close()
	items, p, e := NewFetcher("test", 0).DiscoverSource(context.Background(), Config{}, Source{URL: s.URL + "/1", Kind: "list", List: ListSource{Links: "a.article", Next: "a.next"}}, Progress{}, 5)
	if e == nil || len(items) != 1 || len(p.Pending) != 1 || p.Pending[0] != s.URL+"/2" {
		t.Fatalf("items=%+v progress=%+v err=%v", items, p, e)
	}
}
