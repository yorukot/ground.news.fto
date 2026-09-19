// Package crawl discovers and fetches articles from outlet sites: feeds and
// sitemaps for discovery, robots.txt for permission, a per-host delay for
// politeness, and readability extraction with per-outlet overrides.
package crawl

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// Config is an outlet's crawl_config column.
type Config struct {
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

type Selectors struct {
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
	if cfg.ArticleURLPattern != "" {
		var err error
		if pattern, err = regexp.Compile(cfg.ArticleURLPattern); err != nil {
			return Config{}, nil, fmt.Errorf("article_url_pattern: %w", err)
		}
	}
	return cfg, pattern, nil
}
