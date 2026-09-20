// Package crawl discovers and fetches articles from outlet sites: feeds and
// sitemaps for discovery, robots.txt for permission, a per-host delay for
// politeness, and readability extraction with per-outlet overrides.
package crawl

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"

	"github.com/andybalholm/cascadia"
)

// Config is an outlet's crawl_config column.
type Config struct {
	Lists        []ListSource `json:"lists,omitempty"`
	AllowedHosts []string     `json:"allowed_hosts,omitempty"`
	// Feeds are RSS or Atom URLs.
	Feeds []string `json:"feeds,omitempty"`
	// Sitemaps are news-sitemap URLs, used when an outlet has no usable feed.
	Sitemaps []string `json:"sitemaps,omitempty"`
	// ArticleURLPattern, when set, is a regular expression a discovered URL
	// must match. It keeps out video pages, tag pages and the like.
	ArticleURLPattern string `json:"article_url_pattern,omitempty"`
	// Selectors override readability for sites where it picks the wrong
	// content. Each is a CSS selector; empty means use the default.
	Selectors Selectors `json:"selectors,omitzero"`
}

type ListSource struct {
	Parser        string `json:"parser,omitempty"`
	Item          string `json:"item,omitempty"`
	Date          string `json:"date,omitempty"`
	DateAttribute string `json:"date_attribute,omitempty"`
	NewestFirst   bool   `json:"newest_first,omitempty"`
	URL           string `json:"url"`
	Links         string `json:"links"`
	Next          string `json:"next,omitempty"`
	Category      string `json:"category,omitempty"`
	MaxPages      int    `json:"max_pages,omitempty"`
}

type Selectors struct {
	Lead     string `json:"lead,omitempty"`
	Headline string `json:"headline,omitempty"`
	Body     string `json:"body,omitempty"`
	// Remove lists elements to drop from the body before reading its text.
	Remove []string `json:"remove,omitempty"`
}

func ParseConfig(raw []byte) (Config, *regexp.Regexp, error) {
	var cfg Config
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return Config{}, nil, fmt.Errorf("parse crawl config: %w", err)
		}
	}
	var pattern *regexp.Regexp
	for _, list := range cfg.Lists {
		if list.URL == "" || (list.Links == "" && list.Parser != "ltn") || list.MaxPages < 0 || (list.Parser != "" && list.Parser != "ltn") {
			return Config{}, nil, fmt.Errorf("invalid list source")
		}
		for _, selector := range []string{list.Item, list.Date, list.Links, list.Next} {
			if selector != "" {
				if _, err := cascadia.Compile(selector); err != nil {
					return Config{}, nil, fmt.Errorf("invalid selector: %w", err)
				}
			}
		}
	}
	for _, source := range cfg.Sources() {
		u, err := url.Parse(source.URL)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return Config{}, nil, fmt.Errorf("invalid source URL %q", source.URL)
		}
	}
	for _, selector := range append([]string{cfg.Selectors.Body, cfg.Selectors.Headline, cfg.Selectors.Lead}, cfg.Selectors.Remove...) {
		if selector != "" {
			if _, err := cascadia.Compile(selector); err != nil {
				return Config{}, nil, err
			}
		}
	}
	if cfg.ArticleURLPattern != "" {
		var err error
		if pattern, err = regexp.Compile(cfg.ArticleURLPattern); err != nil {
			return Config{}, nil, fmt.Errorf("article_url_pattern: %w", err)
		}
	}
	return cfg, pattern, nil
}
