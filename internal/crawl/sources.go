package crawl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"
)

// Progress is committed only after the discovered URLs have been stored.
type Progress struct {
	Pending    []string
	Visited    []string
	Done       bool
	Since      time.Time
	StopReason string
}
type Source struct {
	URL, Kind string
	List      ListSource
}

func (c Config) Sources() []Source {
	var out []Source
	for _, u := range c.Feeds {
		out = append(out, Source{URL: u, Kind: "feed"})
	}
	for _, u := range c.Sitemaps {
		out = append(out, Source{URL: u, Kind: "sitemap"})
	}
	for _, l := range c.Lists {
		out = append(out, Source{URL: l.URL, Kind: "list", List: l})
	}
	return out
}

func (f *Fetcher) DiscoverSource(ctx context.Context, cfg Config, src Source, p Progress, limit int) ([]Found, Progress, error) {
	var out []Found
	if p.Done {
		return out, p, nil
	}
	if limit < 1 {
		limit = 5
	}
	if src.Kind == "list" && src.List.MaxPages > 0 {
		limit = min(limit, src.List.MaxPages)
	}
	if src.Kind == "sitemap" {
		limit = min(limit, 10)
	}
	if len(p.Pending) == 0 {
		p.Pending = []string{src.URL}
	}
	allowed := func(raw string) bool {
		u, err := url.Parse(raw)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
			return false
		}
		root, _ := url.Parse(src.URL)
		if strings.EqualFold(root.Host, u.Host) {
			return true
		}
		for _, host := range cfg.AllowedHosts {
			if strings.EqualFold(host, u.Host) {
				return true
			}
		}
		return false
	}
	seen := map[string]bool{}
	for _, u := range p.Visited {
		seen[u] = true
	}
	for pages := 0; len(p.Pending) > 0 && pages < limit; pages++ {
		raw := p.Pending[0]
		if seen[raw] || !allowed(raw) {
			p.Pending = p.Pending[1:]
			continue
		}
		body, final, err := f.Get(ctx, raw)
		if err != nil {
			return out, p, fmt.Errorf("%s: %w", raw, err)
		}
		var entries []Found
		var next []string
		switch src.Kind {
		case "feed":
			feed, err := gofeed.NewParser().Parse(bytes.NewReader(body))
			if err != nil {
				return out, p, err
			}
			for _, i := range feed.Items {
				item := Found{URL: i.Link, Title: i.Title, Category: strings.Join(i.Categories, ",")}
				if i.Image != nil {
					item.ImageURL = i.Image.URL
				}
				if item.ImageURL == "" {
					for _, enclosure := range i.Enclosures {
						if strings.HasPrefix(enclosure.Type, "image/") {
							item.ImageURL = enclosure.URL
							break
						}
					}
				}
				if i.PublishedParsed != nil {
					item.PublishedAt = *i.PublishedParsed
				}
				entries = append(entries, item)
			}
		case "sitemap":
			children := parseSitemapIndex(body)
			if len(children) > 0 {
				sort.SliceStable(children, func(i, j int) bool { return children[i].PublishedAt.After(children[j].PublishedAt) })
				for _, child := range children {
					next = append(next, child.URL)
				}
			} else {
				entries, err = parseSitemap(body)
				if err != nil {
					return out, p, err
				}
			}
		case "list":
			if src.List.Parser == "ltn" {
				entries, next, err = parseLTNList(body, final)
				if err != nil {
					return out, p, err
				}
				break
			}
			doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
			if err != nil {
				return out, p, err
			}
			doc.Find(src.List.Links).Each(func(_ int, s *goquery.Selection) {
				href, ok := s.Attr("href")
				if !ok {
					return
				}
				u, err := final.Parse(href)
				if err == nil {
					item := Found{URL: u.String(), Title: s.Text(), Category: src.List.Category}
					if src.List.Item != "" && src.List.Date != "" {
						date := s.Closest(src.List.Item).Find(src.List.Date).First()
						text := date.Text()
						if src.List.DateAttribute != "" {
							text, _ = date.Attr(src.List.DateAttribute)
						}
						item.PublishedAt = parseTime(text)
					}
					entries = append(entries, item)
				}
			})
			if src.List.Next != "" {
				if href, ok := doc.Find(src.List.Next).First().Attr("href"); ok {
					if u, err := final.Parse(href); err == nil {
						next = append(next, u.String())
					}
				}
			}
		default:
			return out, p, fmt.Errorf("unknown source kind %q", src.Kind)
		}
		older := len(entries) > 0 && !p.Since.IsZero()
		for _, item := range entries {
			if item.PublishedAt.IsZero() || !item.PublishedAt.Before(p.Since) {
				older = false
			}
			item.URL = CanonicalURL(item.URL)
			if item.URL == "" {
				continue
			}
			item.Source = src.URL
			item.Title = cleanText(html.UnescapeString(item.Title))
			if item.Category == "" {
				item.Category = URLCategory(item.URL)
			}
			out = append(out, item)
		}
		if src.Kind == "list" && src.List.NewestFirst && older {
			next = nil
			p.StopReason = "date boundary"
		}
		p.Pending = p.Pending[1:]
		seen[raw] = true
		p.Visited = append(p.Visited, raw)
		for _, u := range next {
			if !seen[u] && allowed(u) {
				p.Pending = append(p.Pending, u)
			}
		}
	}
	p.Done = len(p.Pending) == 0
	if p.Done && p.StopReason == "" {
		p.StopReason = "source exhausted"
	}
	return out, p, nil
}

// LTN's public list endpoint uses numeric object keys in publication order.
func parseLTNList(body []byte, base *url.URL) ([]Found, []string, error) {
	var envelope struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(bytes.TrimPrefix(body, []byte{0xef, 0xbb, 0xbf}), &envelope); err != nil {
		return nil, nil, err
	}
	if envelope.Code != 200 {
		return nil, nil, fmt.Errorf("LTN list code %d", envelope.Code)
	}
	if string(envelope.Data) == "[]" || string(envelope.Data) == "null" {
		return nil, nil, nil
	}
	type listItem struct {
		URL      string `json:"url"`
		Title    string `json:"title"`
		Time     string `json:"time"`
		Category string `json:"type_cn"`
	}
	data := map[string]listItem{}
	if bytes.HasPrefix(bytes.TrimSpace(envelope.Data), []byte("[")) {
		var items []listItem
		if err := json.Unmarshal(envelope.Data, &items); err != nil {
			return nil, nil, err
		}
		for i, item := range items {
			data[strconv.Itoa(i)] = item
		}
	} else if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return nil, nil, err
	}
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { a, _ := strconv.Atoi(keys[i]); b, _ := strconv.Atoi(keys[j]); return a < b })
	var out []Found
	for _, key := range keys {
		i := data[key]
		out = append(out, Found{URL: i.URL, Title: i.Title, Category: i.Category, PublishedAt: parseTime(i.Time)})
	}
	if len(out) == 0 {
		return out, nil, nil
	}
	parts := strings.Split(base.Path, "/")
	n, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return nil, nil, err
	}
	parts[len(parts)-1] = strconv.Itoa(n + 1)
	u := *base
	u.Path = strings.Join(parts, "/")
	return out, []string{u.String()}, nil
}

func URLCategory(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	for _, part := range strings.Split(u.Path, "/") {
		switch part {
		case "politics", "society", "business", "world", "local", "health", "education", "environment", "entertainment", "sports", "shopping", "fashion", "travel":
			return part
		}
	}
	return ""
}

// Explicit section labels only. Unknown sections remain eligible; headlines
// are never used to exclude an article about a public-interest controversy.
func InScope(category string) (bool, string) {
	parts := strings.FieldsFunc(strings.ToLower(category), func(r rune) bool { return r == ',' || r == '/' || r == '|' })
	for _, p := range parts {
		switch strings.TrimSpace(p) {
		case "政治", "社會", "財經", "國際", "地方", "生活", "健康", "教育", "環境", "politics", "society", "business", "world", "local", "health", "education", "environment":
			return true, "public section: " + category
		}
	}
	for _, p := range parts {
		switch strings.TrimSpace(p) {
		case "娛樂", "體育", "消費", "時尚", "旅遊", "entertainment", "sports", "shopping", "fashion", "travel":
			return false, "excluded section: " + category
		}
	}
	return true, "unknown section"
}
