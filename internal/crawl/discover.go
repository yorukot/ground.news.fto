package crawl

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"log/slog"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

// Found is an article URL discovered in a feed or sitemap.
type Found struct {
	URL string
	// Title is the headline as the feed or news sitemap gives it. It is all
	// a headline-only outlet's article ever has.
	Title string
	// PublishedAt is the feed's date, used only if the page itself has none.
	PublishedAt time.Time
}

// maxChildSitemaps bounds how many sitemaps of an index are followed.
const maxChildSitemaps = 2

// Discover lists the article URLs an outlet currently advertises, newest
// first. A failing source is logged and skipped so one broken feed doesn't
// hide the others.
func (f *Fetcher) Discover(ctx context.Context, cfg Config, pattern *regexp.Regexp, purpose Purpose) []Found {
	seen := make(map[string]bool)
	var out []Found
	add := func(raw, title string, at time.Time) {
		clean := CanonicalURL(raw)
		if clean == "" || seen[clean] || (pattern != nil && !pattern.MatchString(clean)) {
			return
		}
		seen[clean] = true
		out = append(out, Found{URL: clean, Title: cleanText(html.UnescapeString(title)), PublishedAt: at})
	}

	for _, feedURL := range cfg.Feeds {
		body, _, err := f.GetFor(ctx, feedURL, purpose)
		if err != nil {
			slog.Warn("feed fetch failed", "feed", feedURL, "error", err)
			continue
		}
		feed, err := gofeed.NewParser().Parse(bytes.NewReader(body))
		if err != nil {
			slog.Warn("feed parse failed", "feed", feedURL, "error", err)
			continue
		}
		for _, item := range feed.Items {
			var at time.Time
			if item.PublishedParsed != nil {
				at = *item.PublishedParsed
			}
			add(item.Link, item.Title, at)
		}
	}

	for _, sitemapURL := range cfg.Sitemaps {
		for _, e := range f.readSitemap(ctx, sitemapURL, purpose, true) {
			add(e.URL, e.Title, e.PublishedAt)
		}
	}

	// Newest first, so the per-crawl cap keeps what is current. Undated
	// entries keep their source order, after the dated ones.
	sort.SliceStable(out, func(i, j int) bool { return out[i].PublishedAt.After(out[j].PublishedAt) })
	return out
}

// readSitemap reads a sitemap, following a sitemap index one level down to
// its most recently modified children.
func (f *Fetcher) readSitemap(ctx context.Context, sitemapURL string, purpose Purpose, followIndex bool) []Found {
	body, _, err := f.GetFor(ctx, sitemapURL, purpose)
	if err != nil {
		slog.Warn("sitemap fetch failed", "sitemap", sitemapURL, "error", err)
		return nil
	}
	if children := parseSitemapIndex(body); len(children) > 0 {
		if !followIndex {
			return nil
		}
		sort.SliceStable(children, func(i, j int) bool { return children[i].PublishedAt.After(children[j].PublishedAt) })
		var out []Found
		for _, child := range children[:min(maxChildSitemaps, len(children))] {
			out = append(out, f.readSitemap(ctx, child.URL, purpose, false)...)
		}
		return out
	}
	entries, err := parseSitemap(body)
	if err != nil {
		slog.Warn("sitemap parse failed", "sitemap", sitemapURL, "error", err)
		return nil
	}
	return entries
}

type sitemapDoc struct {
	URLs []struct {
		Loc     string `xml:"loc"`
		LastMod string `xml:"lastmod"`
		News    struct {
			PublicationDate string `xml:"publication_date"`
			Title           string `xml:"title"`
		} `xml:"news"`
	} `xml:"url"`
}

func parseSitemap(body []byte) ([]Found, error) {
	var doc sitemapDoc
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("decode sitemap: %w", err)
	}
	out := make([]Found, 0, len(doc.URLs))
	for _, u := range doc.URLs {
		at := parseTime(u.News.PublicationDate)
		if at.IsZero() {
			at = parseTime(u.LastMod)
		}
		out = append(out, Found{URL: strings.TrimSpace(u.Loc), Title: u.News.Title, PublishedAt: at})
	}
	return out, nil
}

// parseSitemapIndex returns the child sitemaps of an index, or nil when the
// document is an ordinary sitemap.
func parseSitemapIndex(body []byte) []Found {
	var doc struct {
		XMLName  xml.Name
		Sitemaps []struct {
			Loc     string `xml:"loc"`
			LastMod string `xml:"lastmod"`
		} `xml:"sitemap"`
	}
	if err := xml.Unmarshal(body, &doc); err != nil || doc.XMLName.Local != "sitemapindex" {
		return nil
	}
	out := make([]Found, 0, len(doc.Sitemaps))
	for _, s := range doc.Sitemaps {
		out = append(out, Found{URL: strings.TrimSpace(s.Loc), PublishedAt: parseTime(s.LastMod)})
	}
	return out
}

// trackingParams are dropped so the same article isn't stored once per campaign.
var trackingParams = regexp.MustCompile(`^(utm_.*|fbclid|gclid|from|ref|ocid|_gl)$`)

// CanonicalURL normalizes an article URL: no fragment, no tracking
// parameters, https when the scheme is missing. It returns "" for anything
// that isn't an absolute http(s) URL.
func CanonicalURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return ""
	}
	u.Fragment = ""
	q := u.Query()
	for key := range q {
		if trackingParams.MatchString(strings.ToLower(key)) {
			q.Del(key)
		}
	}
	u.RawQuery = q.Encode()
	u.Host = strings.ToLower(u.Host)
	return u.String()
}
