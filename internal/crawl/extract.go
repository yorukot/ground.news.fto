package crawl

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	readability "codeberg.org/readeck/go-readability/v2"
	"github.com/PuerkitoBio/goquery"
)

// minBodyRunes rejects pages where extraction clearly failed (paywalls,
// video-only pages, consent walls).
const minBodyRunes = 120

// ErrNoArticle means the page has no extractable article text.
var ErrNoArticle = errors.New("no article content found")

type Article struct {
	Headline    string
	Body        string
	PublishedAt time.Time
	ImageURL    string
	// Provider is the original publisher named in the page's metadata. It
	// matters on aggregators, where the article is credited to that outlet.
	Provider string
}

// Extract pulls the headline, plain-text body and publish time out of an
// article page. Selectors, when configured, take priority over readability.
func Extract(page []byte, pageURL *url.URL, sel Selectors) (Article, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(page))
	if err != nil {
		return Article{}, fmt.Errorf("parse html: %w", err)
	}
	meta := readMetadata(doc)

	var art Article
	art.PublishedAt = meta.published
	art.Provider = meta.provider
	art.ImageURL = resolveImageURL(meta.image, pageURL)

	if sel.Headline != "" {
		art.Headline = cleanText(doc.Find(sel.Headline).First().Text())
	}
	if sel.Body != "" {
		body := doc.Find(sel.Body).First()
		for _, remove := range sel.Remove {
			body.Find(remove).Remove()
		}
		body.Find("script, style, figure, iframe, noscript").Remove()
		art.Body = paragraphs(body)
	}

	if art.Headline == "" || art.Body == "" {
		readable, err := readability.FromReader(bytes.NewReader(page), pageURL)
		if err == nil {
			if art.Headline == "" {
				art.Headline = cleanText(readable.Title())
			}
			if art.Body == "" && readable.Node != nil {
				var text bytes.Buffer
				if err := readable.RenderText(&text); err == nil {
					art.Body = normalizeBody(text.String())
				}
			}
			if art.PublishedAt.IsZero() {
				if at, err := readable.PublishedTime(); err == nil {
					art.PublishedAt = at
				}
			}
		}
	}
	// Without a headline selector, og:title beats <title>, and either may
	// carry a " | Section | Site name" suffix that isn't part of the headline.
	if sel.Headline == "" || art.Headline == "" {
		if meta.title != "" {
			art.Headline = meta.title
		}
		art.Headline = stripSiteSuffix(art.Headline)
	}
	art.Headline = stripSiteName(art.Headline, meta.siteName)

	art.Body = trimRelated(dropFurniture(art.Body, art.Headline))

	if art.Headline == "" || len([]rune(art.Body)) < minBodyRunes {
		return Article{}, ErrNoArticle
	}
	return art, nil
}

// timestampLine matches a line that is only a publish or update time.
var timestampLine = regexp.MustCompile(`^(更新時間|發布時間|發稿時間|發佈時間)|^\d{4}[./-]\d{1,2}[./-]\d{1,2}\s+\d{1,2}:\d{2}`)

// dropFurniture removes page furniture that readability sometimes keeps with
// the article: a repeat of the headline, bare timestamps, and one- or
// two-character labels such as a "最新" tab.
func dropFurniture(body, headline string) string {
	paragraphs := strings.Split(body, "\n\n")
	kept := paragraphs[:0]
	for _, p := range paragraphs {
		if p == headline || timestampLine.MatchString(p) || len([]rune(p)) <= 2 {
			continue
		}
		kept = append(kept, p)
	}
	return strings.Join(kept, "\n\n")
}

// relatedMarkers open the "read more" blocks outlets append to an article.
// They are other articles' headlines, and must not be summarized as if they
// were part of this one.
var relatedMarkers = []string{
	"更多新聞", "延伸閱讀", "相關新聞", "看更多", "更多報導", "推薦閱讀", "你可能也想看",
	"熱門新聞", "其他人也在看", "更多內容", "◤", "►", "▶", "【延伸閱讀】", "《更多",
}

// trimRelated cuts the body at the first paragraph that opens such a block,
// provided it comes late enough that real content isn't lost to a stray match.
func trimRelated(body string) string {
	paragraphs := strings.Split(body, "\n\n")
	for i, p := range paragraphs {
		if i < len(paragraphs)/2 {
			continue
		}
		for _, marker := range relatedMarkers {
			if strings.HasPrefix(p, marker) {
				return strings.Join(paragraphs[:i], "\n\n")
			}
		}
	}
	return body
}

// stripSiteSuffix cuts a headline at the first " | " or " ｜ ". Outlets use
// the bar only to append section and site names; a dash is left alone because
// headlines themselves contain dashes.
func stripSiteSuffix(headline string) string {
	for _, sep := range []string{" | ", " ｜ ", "｜"} {
		if i := strings.Index(headline, sep); i > 0 {
			headline = headline[:i]
		}
	}
	return strings.TrimSpace(headline)
}

// stripSiteName removes a trailing " - Site name". A dash is only cut when
// what follows is the page's own declared site name, because headlines
// themselves contain dashes.
func stripSiteName(headline, siteName string) string {
	if siteName == "" {
		return headline
	}
	for _, sep := range []string{" - ", " – ", " — ", "－"} {
		if rest, ok := strings.CutSuffix(headline, sep+siteName); ok {
			return strings.TrimSpace(rest)
		}
	}
	return headline
}

type metadata struct {
	siteName  string
	title     string
	image     string
	published time.Time
	provider  string
}

func readMetadata(doc *goquery.Document) metadata {
	var m metadata
	attr := func(selector string) string {
		v, _ := doc.Find(selector).First().Attr("content")
		return strings.TrimSpace(v)
	}
	m.title = cleanText(attr(`meta[property="og:title"]`))
	m.siteName = cleanText(attr(`meta[property="og:site_name"]`))
	for _, selector := range []string{
		`meta[property="og:image:secure_url"]`,
		`meta[property="og:image"]`,
		`meta[name="twitter:image"]`,
		`meta[property="twitter:image"]`,
	} {
		if m.image = attr(selector); m.image != "" {
			break
		}
	}
	for _, selector := range []string{
		`meta[property="article:published_time"]`,
		`meta[name="pubdate"]`,
		`meta[name="date"]`,
		`meta[itemprop="datePublished"]`,
	} {
		if at := parseTime(attr(selector)); !at.IsZero() {
			m.published = at
			break
		}
	}

	// JSON-LD carries the publish date and, on aggregators, the real publisher.
	doc.Find(`script[type="application/ld+json"]`).Each(func(_ int, s *goquery.Selection) {
		var raw any
		if json.Unmarshal([]byte(s.Text()), &raw) != nil {
			return
		}
		for _, node := range flattenLD(raw) {
			if m.image == "" {
				m.image = imageFromLD(node["image"])
			}
			if m.published.IsZero() {
				if v, ok := node["datePublished"].(string); ok {
					m.published = parseTime(v)
				}
			}
			if m.provider == "" {
				for _, key := range []string{"provider", "publisher", "sourceOrganization"} {
					if org, ok := node[key].(map[string]any); ok {
						if name, ok := org["name"].(string); ok && name != "" {
							m.provider = strings.TrimSpace(name)
							break
						}
					}
				}
			}
		}
	})
	return m
}

// ExtractImage reads only public page metadata. It is used by the maintenance
// backfill and never returns article text.
func ExtractImage(page []byte, pageURL *url.URL) string {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(page))
	if err != nil {
		return ""
	}
	return resolveImageURL(readMetadata(doc).image, pageURL)
}

func imageFromLD(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case []any:
		for _, item := range v {
			if image := imageFromLD(item); image != "" {
				return image
			}
		}
	case map[string]any:
		for _, key := range []string{"url", "contentUrl"} {
			if image, ok := v[key].(string); ok && strings.TrimSpace(image) != "" {
				return strings.TrimSpace(image)
			}
		}
	}
	return ""
}

func resolveImageURL(raw string, base *url.URL) string {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || raw == "" {
		return ""
	}
	if base != nil {
		u = base.ResolveReference(u)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ""
	}
	u.Fragment = ""
	return u.String()
}

func flattenLD(v any) []map[string]any {
	switch t := v.(type) {
	case []any:
		var out []map[string]any
		for _, item := range t {
			out = append(out, flattenLD(item)...)
		}
		return out
	case map[string]any:
		out := []map[string]any{t}
		if graph, ok := t["@graph"]; ok {
			out = append(out, flattenLD(graph)...)
		}
		return out
	}
	return nil
}

func paragraphs(sel *goquery.Selection) string {
	var parts []string
	sel.Find("p").Each(func(_ int, p *goquery.Selection) {
		if text := cleanText(p.Text()); text != "" && !isCaption(text) {
			parts = append(parts, text)
		}
	})
	if len(parts) == 0 {
		return normalizeBody(sel.Text())
	}
	return strings.Join(parts, "\n\n")
}

// isCaption reports a photo caption. Taiwanese outlets mark them with a
// leading triangle, and they describe the picture, not the news.
func isCaption(line string) bool {
	return strings.HasPrefix(line, "▲") || strings.HasPrefix(line, "▼")
}

func normalizeBody(text string) string {
	var parts []string
	for _, line := range strings.Split(text, "\n") {
		if line = cleanText(line); line != "" && !isCaption(line) {
			parts = append(parts, line)
		}
	}
	return strings.Join(parts, "\n\n")
}

func cleanText(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

var timeLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05Z0700",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006/01/02 15:04:05",
	"2006/01/02 15:04",
	"2006-01-02 15:04",
	"2006-01-02",
}

var taipei = time.FixedZone("Asia/Taipei", 8*60*60)

// parseTime reads the date formats outlets actually use. A time without a
// zone is taken as Taiwan time, which is what Taiwanese outlets mean.
func parseTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	for _, layout := range timeLayouts {
		if at, err := time.ParseInLocation(layout, s, taipei); err == nil {
			return at
		}
	}
	return time.Time{}
}
