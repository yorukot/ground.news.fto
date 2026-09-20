// Package config reads the process configuration from the environment once at startup.
package config

import (
	"errors"
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL string
	HTTPAddr    string
	// AdminToken guards /api/v1/admin/*. Admin endpoints are disabled when empty.
	AdminToken string

	// OpenAIAPIKey selects the real AI client. When empty AI dispatch pauses unless explicitly enabled for development.
	OpenAIAPIKey string
	// OpenAIBaseURL points at any OpenAI-compatible chat-completions endpoint
	// (for example a LiteLLM proxy). Empty means api.openai.com.
	OpenAIBaseURL      string
	OpenAISummaryModel string

	// CrawlerUserAgent identifies the crawler to outlets and is the name
	// matched against their robots.txt. It should say who to contact.
	CrawlerUserAgent string
	// DailyAnalysisLimit caps new articles admitted to AI per Taiwan calendar day.
	DailyAnalysisLimit int
	AllowFakeAI        bool
}

func Load() (Config, error) {
	c := Config{
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		HTTPAddr:           envOr("HTTP_ADDR", "127.0.0.1:8080"),
		AdminToken:         os.Getenv("ADMIN_TOKEN"),
		OpenAIAPIKey:       os.Getenv("OPENAI_API_KEY"),
		OpenAIBaseURL:      os.Getenv("OPENAI_BASE_URL"),
		OpenAISummaryModel: os.Getenv("OPENAI_SUMMARY_MODEL"),
		CrawlerUserAgent:   envOr("CRAWLER_USER_AGENT", "GroundNewsTW-bot/0.1"),
	}
	c.AllowFakeAI = os.Getenv("ALLOW_FAKE_AI") == "true"
	c.DailyAnalysisLimit = 500
	if raw := os.Getenv("AI_MAX_NEW_ARTICLES_PER_DAY"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			return Config{}, errors.New("AI_MAX_NEW_ARTICLES_PER_DAY must be a positive integer")
		}
		c.DailyAnalysisLimit = n
	}
	if c.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if c.OpenAIAPIKey != "" && c.OpenAISummaryModel == "" {
		return Config{}, errors.New("OPENAI_SUMMARY_MODEL is required when OPENAI_API_KEY is set")
	}
	return c, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
