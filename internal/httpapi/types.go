package httpapi

import "time"

// These types are the JSON contract described in api/openapi.yaml.
// Article bodies are deliberately absent: article text is never served.

type Outlet struct {
	ID     int64  `json:"id"`
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	Domain string `json:"domain,omitempty"`
}

type EventSummary struct {
	ID                int64     `json:"id"`
	Title             string    `json:"title"`
	FirstSeenAt       time.Time `json:"firstSeenAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
	ArticleCount      int64     `json:"articleCount"`
	OutletCount       int64     `json:"outletCount"`
	LatestDevelopment string    `json:"latestDevelopment"`
}

type EventsPage struct {
	Events     []EventSummary `json:"events"`
	NextCursor *string        `json:"nextCursor"`
}

type EventDetail struct {
	ID           int64          `json:"id"`
	Title        string         `json:"title"`
	FirstSeenAt  time.Time      `json:"firstSeenAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	ArticleCount int64          `json:"articleCount"`
	OutletCount  int64          `json:"outletCount"`
	Timeline     []TimelineStep `json:"timeline"`
	Articles     []Article      `json:"articles"`
}

// TimelineStep is one development in the event, oldest first.
type TimelineStep struct {
	// ArticleID is the first article that reported the development.
	ArticleID int64 `json:"articleId"`
	// Development is one short neutral line in the site's own voice.
	Development string `json:"development"`
	// HappenedOn is a calendar date (YYYY-MM-DD) in Asia/Taipei.
	HappenedOn        string       `json:"happenedOn"`
	DateIsApproximate bool         `json:"dateIsApproximate"`
	Reports           []StepReport `json:"reports"`
}

// StepReport is one outlet's report of a development, ordered by publish time.
type StepReport struct {
	// ArticleID is the article card to jump to; for a wire reprint it is the original.
	ArticleID   int64     `json:"articleId"`
	Outlet      Outlet    `json:"outlet"`
	PublishedAt time.Time `json:"publishedAt"`
}

type Article struct {
	ID          int64     `json:"id"`
	Outlet      Outlet    `json:"outlet"`
	Headline    string    `json:"headline"`
	URL         string    `json:"url"`
	PublishedAt time.Time `json:"publishedAt"`
	Summary     string    `json:"summary"`
	// HeadlineOnly is true for outlets that don't allow AI use of their
	// content: the site lists the headline and link, and never a summary.
	HeadlineOnly bool `json:"headlineOnly"`
	// Reprints lists the other outlets that ran the same wire story.
	Reprints []Reprint `json:"reprints"`
}

type Reprint struct {
	Outlet      Outlet    `json:"outlet"`
	URL         string    `json:"url"`
	PublishedAt time.Time `json:"publishedAt"`
}

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
