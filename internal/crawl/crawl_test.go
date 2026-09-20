package crawl

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestCanonicalURL(t *testing.T) {
	cases := map[string]string{
		"https://Example.com/news/1?utm_source=fb&id=7#top": "https://example.com/news/1?id=7",
		"https://example.com/a?fbclid=x":                    "https://example.com/a",
		"/relative/path":                                    "",
		"javascript:void(0)":                                "",
	}
	for in, want := range cases {
		if got := CanonicalURL(in); got != want {
			t.Errorf("CanonicalURL(%q) = %q, want %q", in, got, want)
		}
	}
}

const page = `<!doctype html><html><head>
<title>範例標題 | 範例新聞網</title>
<meta property="og:title" content="範例標題">
<meta property="article:published_time" content="2026-09-19T08:30:00+08:00">
<script type="application/ld+json">{"@type":"NewsArticle","datePublished":"2026-09-19T08:30:00+08:00","provider":{"@type":"Organization","name":"中央社"}}</script>
</head><body>
<nav>首頁 政治 社會</nav>
<article><h1>範例標題</h1>
<div class="story">
<p>範例市政府今天將明年度總預算案送交市議會審議，歲出規模創歷年新高，市府表示增加的經費主要用於社會福利與公共運輸。</p>
<p class="ad">廣告：請訂閱我們的電子報</p>
<p>市長受訪時表示，這份預算務實穩健，盼議會支持；多名議員則表示將嚴格審查舉債額度，並要求市府說明各項新增計畫的必要性。</p>
<p>議會預計下週開始分組審查，全案最快下月底完成三讀，屆時市府將依審議結果調整各局處的年度施政計畫。</p>
</div></article>
<footer>版權所有</footer></body></html>`

func TestExtract(t *testing.T) {
	u, _ := url.Parse("https://example.com/news/1")

	t.Run("readability default", func(t *testing.T) {
		art, err := Extract([]byte(page), u, Selectors{})
		if err != nil {
			t.Fatal(err)
		}
		if art.Headline != "範例標題" {
			t.Errorf("headline = %q", art.Headline)
		}
		if !strings.Contains(art.Body, "總預算案") || strings.Contains(art.Body, "版權所有") {
			t.Errorf("body = %q", art.Body)
		}
		want := time.Date(2026, 9, 19, 0, 30, 0, 0, time.UTC)
		if !art.PublishedAt.Equal(want) {
			t.Errorf("published = %v, want %v", art.PublishedAt, want)
		}
		if art.Provider != "中央社" {
			t.Errorf("provider = %q", art.Provider)
		}
	})

	t.Run("selector override removes unwanted elements", func(t *testing.T) {
		art, err := Extract([]byte(page), u, Selectors{Body: ".story", Remove: []string{".ad"}})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(art.Body, "廣告") {
			t.Errorf("ad paragraph should be removed: %q", art.Body)
		}
	})

	t.Run("page without an article", func(t *testing.T) {
		_, err := Extract([]byte(`<html><head><title>x</title></head><body><p>短</p></body></html>`), u, Selectors{})
		if !errors.Is(err, ErrNoArticle) {
			t.Fatalf("err = %v, want ErrNoArticle", err)
		}
	})
}

func TestStripSiteName(t *testing.T) {
	if got := stripSiteName("蘇巧慧駁斥耍特權傳言 - 民視新聞網", "民視新聞網"); got != "蘇巧慧駁斥耍特權傳言" {
		t.Errorf("got %q", got)
	}
	// A dash that is part of the headline stays.
	if got := stripSiteName("颱風逼近 - 各地停班停課一覽", "民視新聞網"); got != "颱風逼近 - 各地停班停課一覽" {
		t.Errorf("got %q", got)
	}
}

func TestDropFurniture(t *testing.T) {
	body := "2026.09.19 12:01 臺北時間\n\n範例標題\n\n最新\n\n第一段內文，說明事件經過。\n\n更新時間｜2026.09.19 12:01 臺北時間"
	if got := dropFurniture(body, "範例標題"); got != "第一段內文，說明事件經過。" {
		t.Errorf("dropFurniture = %q", got)
	}
}

func TestTrimRelated(t *testing.T) {
	body := "第一段內文。\n\n第二段內文。\n\n第三段內文。\n\n更多新聞： 另一則報導的標題\n\n又一則報導的標題"
	if got := trimRelated(body); got != "第一段內文。\n\n第二段內文。\n\n第三段內文。" {
		t.Errorf("trimRelated = %q", got)
	}
	// A marker in the opening half is part of the article, not a trailing block.
	early := "延伸閱讀是一種習慣。\n\n第二段。\n\n第三段。\n\n第四段。"
	if got := trimRelated(early); got != early {
		t.Errorf("an early marker must be kept: %q", got)
	}
}

func TestFetcherRespectsRobotsAndDelay(t *testing.T) {
	var hits []time.Time
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			w.Write([]byte("User-agent: *\nDisallow: /private/\n"))
		default:
			hits = append(hits, time.Now())
			w.Write([]byte("ok"))
		}
	}))
	defer srv.Close()

	f := NewFetcher("TestBot/1.0", 150*time.Millisecond)
	ctx := context.Background()

	if _, _, err := f.Get(ctx, srv.URL+"/private/page"); !errors.Is(err, ErrDisallowed) {
		t.Fatalf("err = %v, want ErrDisallowed", err)
	}
	for range 2 {
		if _, _, err := f.Get(ctx, srv.URL+"/news/1"); err != nil {
			t.Fatal(err)
		}
	}
	if len(hits) != 2 {
		t.Fatalf("hits = %d, want 2 (the disallowed page must not be requested)", len(hits))
	}
	if gap := hits[1].Sub(hits[0]); gap < 140*time.Millisecond {
		t.Errorf("requests to one host were %v apart, want at least the host delay", gap)
	}
}

func TestContentSignalDoesNotDisableExtraction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			w.Write([]byte("User-agent: *\nContent-Signal: ai-input=no\nDisallow: /private\n"))
			return
		}
		w.Write([]byte(page))
	}))
	defer srv.Close()
	f := NewFetcher("TestBot/1.0", 0)
	if _, _, err := f.Get(context.Background(), srv.URL+"/news"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.Get(context.Background(), srv.URL+"/private"); !errors.Is(err, ErrDisallowed) {
		t.Fatal(err)
	}
}

func TestParseSitemap(t *testing.T) {
	body := `<?xml version="1.0"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:news="http://www.google.com/schemas/sitemap-news/0.9">
<url><loc>https://example.com/n/1</loc><news:news><news:publication_date>2026-09-19T08:00:00+08:00</news:publication_date><news:title>範例標題 &amp; 副題</news:title></news:news></url>
<url><loc>https://example.com/n/2</loc><lastmod>2026-09-18</lastmod></url></urlset>`
	got, err := parseSitemap([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].URL != "https://example.com/n/1" || got[0].PublishedAt.IsZero() || !got[1].PublishedAt.IsZero() {
		t.Fatalf("got %+v", got)
	}
	if got[0].Title != "範例標題 & 副題" {
		t.Errorf("title = %q", got[0].Title)
	}

	index := `<?xml version="1.0"?><sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
<sitemap><loc>https://example.com/s/old.xml</loc><lastmod>2026-01-01</lastmod></sitemap>
<sitemap><loc>https://example.com/s/new.xml</loc><lastmod>2026-09-19</lastmod></sitemap></sitemapindex>`
	if children := parseSitemapIndex([]byte(index)); len(children) != 2 || children[1].URL != "https://example.com/s/new.xml" {
		t.Fatalf("index children = %+v", children)
	}
	if parseSitemapIndex([]byte(body)) != nil {
		t.Error("an ordinary sitemap is not an index")
	}
}
