package seed

import (
	"github.com/yorukot/ground-news-tw/internal/crawl"
	"net/url"
	"strings"
	"testing"
)

// Synthetic text in the observed publisher DOM structures: verify boundaries
// and removed furniture without copying publishers' articles into fixtures.
func TestPublisherExtractionBoundaries(t *testing.T) {
	text := strings.Repeat("市議會今天審查公共運輸預算，市府說明執行進度，議員要求公開資料。", 6)
	for _, tc := range []struct{ slug, html string }{
		{"udn", `<h1 class="article-content__title">新聞標題</h1><section class="article-content__editor"><p>FIRST` + text + `LAST</p><div class="udn-ads"><p>ADVERT</p></div></section><p>會員規範</p>`},
		{"ltn", `<div class="article"><h1>新聞標題</h1><div class="text boxText"><div class="photo"><p>CAPTION</p></div><p>FIRST` + text + `LAST</p><p class="appE1121">ADVERT</p></div></div>`},
		{"chinatimes", `<h1 class="article-title">新聞標題</h1><div class="article-body"><p>FIRST` + text + `LAST</p><div class="promote-word"><p>ADVERT</p></div></div><p>RELATED</p>`},
		{"pts", `<h1 class="article-title">新聞標題</h1><div class="post-article"><div class="articleimg">FIRST</div><p>` + text + `LAST</p></div>`},
		{"cna", `<div class="centralContent"><h1>新聞標題</h1><div class="paragraph"><p>FIRST` + text + `LAST</p></div><div class="paragraph appDownload"><p>ADVERT</p></div></div>`},
		{"ettoday", `<h1 class="title">新聞標題</h1><div class="story" itemprop="articleBody"><p>FIRST` + text + `LAST</p><div class="ad_readmore"><p>ADVERT</p></div></div>`},
		{"setn", `<h1>新聞標題</h1><div id="newsContent"><p>FIRST` + text + `LAST</p><figure><p>CAPTION</p></figure></div>`},
		{"cti", `<h1 class="article-title">新聞標題</h1><div class="rendered-content"><p>FIRST` + text + `LAST</p><figure><p>CAPTION</p></figure></div>`},
	} {
		t.Run(tc.slug, func(t *testing.T) {
			for _, o := range Outlets {
				if o.Slug != tc.slug {
					continue
				}
				u, _ := url.Parse("https://" + o.Domain + "/article")
				art, err := crawl.Extract([]byte("<html><body>"+tc.html+"</body></html>"), u, o.Crawl.Selectors)
				if err != nil {
					t.Fatal(err)
				}
				if art.Headline != "新聞標題" || !strings.HasPrefix(art.Body, "FIRST") || !strings.HasSuffix(art.Body, "LAST") || strings.Contains(art.Body, "ADVERT") || strings.Contains(art.Body, "CAPTION") {
					t.Fatalf("bad extraction: %+v", art)
				}
				return
			}
			t.Fatal("missing outlet")
		})
	}
}
