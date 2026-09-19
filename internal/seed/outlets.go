package seed

import "github.com/yorukot/ground-news-tw/internal/crawl"

// Outlet is a news source to register. An outlet is crawled only when it has
// a Crawl config, and it gets one only after `app crawl-check <slug>` has
// shown that its robots.txt allows us and that extraction gives clean text.
type Outlet struct {
	Slug         string
	Name         string
	Domain       string
	IsAggregator bool
	// Coverage is CoverageFull unless set. See docs/outlets.md.
	Coverage string
	// Crawl is nil for outlets that are registered but not crawled.
	Crawl *crawl.Config
	// NotCrawled says why an outlet's articles are not given to the model: either
	// it is not crawled at all, or it is in headline-only coverage.
	NotCrawled string
}

const (
	// CoverageFull: article pages are fetched and given to the model.
	CoverageFull = "full"
	// CoverageHeadline: only feed metadata (headline, link, time) is kept.
	// The page is never fetched and nothing reaches a model.
	CoverageHeadline = "headline"
)

// Reasons an outlet with a working feed is still not crawled. Summarizing an
// article means giving it to an AI model, so a publisher's stated position on
// AI use is respected even where robots.txt rules would let a crawler in.
// Checked 2026-09-19; re-check before changing any of these.
const (
	refusesAIUse    = "robots.txt states its content may not be used for AI or LLM purposes"
	refusesAIInput  = "robots.txt Content-Signal says ai-input=no"
	blocksAICrawler = "robots.txt allows * but blocks every named AI crawler; needs a decision or permission"
)

// Outlets is the starting list from plan.md.
var Outlets = []Outlet{
	{
		// robots.txt: Content-signal ai-input=yes, ai-train=no.
		Slug: "cna", Name: "中央社", Domain: "cna.com.tw",
		Crawl: &crawl.Config{
			Feeds: []string{
				"https://feeds.feedburner.com/rsscna/politics",
				"https://feeds.feedburner.com/rsscna/social",
				"https://feeds.feedburner.com/rsscna/finance",
				"https://feeds.feedburner.com/rsscna/lifehealth",
				"https://feeds.feedburner.com/rsscna/local",
			},
			ArticleURLPattern: `^https://www\.cna\.com\.tw/news/[a-z]+/\d+\.aspx$`,
			Selectors: crawl.Selectors{
				Headline: "div.centralContent h1",
				Body:     "div.centralContent div.paragraph:not(.appDownload):not(.articleADbox):not(.bottomArticleBanner):not(.moreArticle)",
			},
		},
	},
	{
		Slug: "pts", Name: "公視新聞網", Domain: "news.pts.org.tw",
		Crawl: &crawl.Config{
			Feeds:             []string{"https://news.pts.org.tw/xml/newsfeed.xml"},
			ArticleURLPattern: `^https://news\.pts\.org\.tw/article/\d+$`,
			Selectors:         crawl.Selectors{Headline: "h1.article-title", Body: "div.post-article"},
		},
	},
	{Slug: "udn", Name: "聯合新聞網", Domain: "udn.com", Coverage: CoverageHeadline, NotCrawled: refusesAIUse, Crawl: &crawl.Config{Sitemaps: []string{"https://udn.com/sitemap/gnews/2"}, ArticleURLPattern: `^https://udn\.com/news/story/\d+/\d+$`}},
	{Slug: "ltn", Name: "自由時報", Domain: "ltn.com.tw", Coverage: CoverageHeadline, NotCrawled: blocksAICrawler, Crawl: &crawl.Config{Feeds: []string{"https://news.ltn.com.tw/rss/all.xml"}, ArticleURLPattern: `^https://news\.ltn\.com\.tw/news/(politics|society|life|world|business|local)/`}},
	{Slug: "chinatimes", Name: "中時新聞網", Domain: "chinatimes.com", Coverage: CoverageHeadline, NotCrawled: blocksAICrawler, Crawl: &crawl.Config{Sitemaps: []string{"https://www.chinatimes.com/sitemaps/sitemap_todaynews.xml"}, ArticleURLPattern: `^https://www\.chinatimes\.com/(realtimenews|newspapers)/\d+-26(0402|0405|0407|0408|0409|0410)$`}},
	{
		Slug: "ettoday", Name: "ETtoday新聞雲", Domain: "ettoday.net",
		Crawl: &crawl.Config{
			Feeds:             []string{"https://feeds.feedburner.com/ettoday/news"},
			ArticleURLPattern: `^https://www\.ettoday\.net/news/\d+/\d+\.htm$`,
			Selectors: crawl.Selectors{
				Headline: "h1.title",
				Body:     `div.story[itemprop="articleBody"]`,
				Remove:   []string{".et_editor", ".ad_in_news", ".ad_readmore", "p.no_margin", ".et_social_1", ".et_social_2"},
			},
		},
	},
	{Slug: "setn", Name: "三立新聞網", Domain: "setn.com", Crawl: &crawl.Config{Sitemaps: []string{"https://www.setn.com/sitemapGoogleNews.xml"}, ArticleURLPattern: `^https://www\.setn\.com/news/\d+$`, Selectors: crawl.Selectors{Headline: "h1", Body: "div#newsContent"}}},
	{Slug: "tvbs", Name: "TVBS新聞網", Domain: "news.tvbs.com.tw", Crawl: &crawl.Config{Sitemaps: []string{"https://news.tvbs.com.tw/sitemap/news-sitemap"}, ArticleURLPattern: `^https://news\.tvbs\.com\.tw/(politics|local|world|life|money|china|focus)/\d+$`}},
	{Slug: "ebc", Name: "東森新聞", Domain: "news.ebc.net.tw", Crawl: &crawl.Config{Sitemaps: []string{"https://news.ebc.net.tw/sitemap/realtime.xml"}, ArticleURLPattern: `^https://news\.ebc\.net\.tw/news/(politics|society|living|world|business|china)/\d+$`}},
	{Slug: "ftv", Name: "民視新聞網", Domain: "ftvnews.com.tw", Crawl: &crawl.Config{Sitemaps: []string{"https://www.ftvnews.com.tw/sitemap/sitemap.xml"}, ArticleURLPattern: `^https://www\.ftvnews\.com\.tw/news/detail/\w+$`}},
	{Slug: "cti", Name: "中天新聞網", Domain: "ctinews.com", Crawl: &crawl.Config{Sitemaps: []string{"https://ctinews.com/rss/sitemap-news.xml"}, ArticleURLPattern: `^https://ctinews\.com/news/items/\w+$`, Selectors: crawl.Selectors{Headline: "h1.article-title", Body: "div.rendered-content"}}},
	{Slug: "mirrormedia", Name: "鏡週刊", Domain: "mirrormedia.mg", Crawl: &crawl.Config{Feeds: []string{"https://www.mirrormedia.mg/rss/rss.xml"}, ArticleURLPattern: `^https://www\.mirrormedia\.mg/story/\w+$`}},
	{Slug: "storm", Name: "風傳媒", Domain: "storm.mg", Crawl: &crawl.Config{Sitemaps: []string{"https://www.storm.mg/sitemaps/1/sitemap.xml"}, ArticleURLPattern: `^https://www\.storm\.mg/article/\d+$`}},
	{Slug: "upmedia", Name: "上報", Domain: "upmedia.mg", Coverage: CoverageHeadline, NotCrawled: blocksAICrawler, Crawl: &crawl.Config{Sitemaps: []string{"https://www.upmedia.mg/sitemapnews"}, ArticleURLPattern: `^https://www\.upmedia\.mg/tw/(focus|investigation|commentary)/`}},
	{Slug: "newtalk", Name: "Newtalk新聞", Domain: "newtalk.tw", Coverage: CoverageHeadline, NotCrawled: refusesAIInput, Crawl: &crawl.Config{Feeds: []string{"https://newtalk.tw/rss/all"}}},
	{Slug: "nownews", Name: "NOWnews今日新聞", Domain: "nownews.com", Crawl: &crawl.Config{Sitemaps: []string{"https://www.nownews.com/newsSitemap-daily.xml"}, ArticleURLPattern: `^https://www\.nownews\.com/news/\d+$`}},
	{Slug: "thenewslens", Name: "關鍵評論網", Domain: "thenewslens.com", Coverage: CoverageHeadline, NotCrawled: blocksAICrawler, Crawl: &crawl.Config{Feeds: []string{"https://feeds.feedburner.com/TheNewsLens"}}},
	{Slug: "twreporter", Name: "報導者", Domain: "twreporter.org", Coverage: CoverageHeadline, NotCrawled: refusesAIUse + "; it says licensing may be available", Crawl: &crawl.Config{Feeds: []string{"https://www.twreporter.org/a/rss2.xml"}}},
	{Slug: "cw", Name: "天下雜誌", Domain: "cw.com.tw", NotCrawled: "allowed, but its RSS feed is dead (latest item 2021) and robots.txt lists no sitemap"},
	{Slug: "yahoo", Name: "Yahoo奇摩新聞", Domain: "tw.news.yahoo.com", IsAggregator: true},
}
