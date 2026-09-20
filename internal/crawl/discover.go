package crawl

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Found is an article URL discovered in a feed or sitemap.
type Found struct {
	URL      string
	Source   string
	Category string
	// Title is discovery metadata; the article page may provide a cleaner title.
	Title string
	// ImageURL is an image advertised by the feed or news sitemap.
	ImageURL string
	// PublishedAt is the feed's date, used only if the page itself has none.
	PublishedAt time.Time
}

type sitemapDoc struct {
	URLs []struct {
		Loc     string `xml:"loc"`
		LastMod string `xml:"lastmod"`
		News    struct {
			PublicationDate string `xml:"publication_date"`
			Title           string `xml:"title"`
		} `xml:"news"`
		Images []struct {
			Loc string `xml:"loc"`
		} `xml:"image"`
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
		imageURL := ""
		if len(u.Images) > 0 {
			imageURL = strings.TrimSpace(u.Images[0].Loc)
		}
		out = append(out, Found{
			URL: strings.TrimSpace(u.Loc), Title: u.News.Title,
			ImageURL: imageURL, PublishedAt: at,
		})
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
