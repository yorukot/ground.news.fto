// Package ai is the only place that talks to a language model. The rest of
// the code depends on the Client interface, so another provider can be added
// without touching the pipeline.
package ai

import "context"

// PromptVersion is recorded on every summary. Bump it whenever a prompt file
// changes in a way that could change outputs.
const PromptVersion = "v3"

type Client interface {
	// SummarizeArticle analyzes one article on its own: no other article's
	// facts may leak into the result.
	SummarizeArticle(ctx context.Context, in ArticleInput) (ArticleAnalysis, error)
	// LinkToEvent picks the candidate event the article belongs to, or none.
	LinkToEvent(ctx context.Context, in LinkInput) (LinkDecision, error)
	// TitleEvent writes a short factual title for a new event.
	TitleEvent(ctx context.Context, in TitleInput) (string, error)
	// Translate writes Traditional Chinese versions of the site's own English
	// text. It backfills older rows and titles new events; new article
	// summaries are written in Chinese directly from the source article.
	Translate(ctx context.Context, in TranslateInput) (Translation, error)
	// Model names the model behind this client, for the summaries table.
	Model() string
}

type ArticleInput struct {
	Outlet      string
	Headline    string
	Body        string
	PublishedOn string // YYYY-MM-DD in Asia/Taipei, so "yesterday" can be resolved
}

type EntityKind string

const (
	KindPerson       EntityKind = "person"
	KindOrganization EntityKind = "organization"
	KindPlace        EntityKind = "place"
)

type Entity struct {
	// Name is the entity as written in the article, in the original language.
	// Names are matched across articles, and the original script is the one
	// form every outlet shares.
	Name    string     `json:"name"`
	Kind    EntityKind `json:"kind"`
	Aliases []string   `json:"aliases"`
}

type ArticleAnalysis struct {
	// Summary is 2–4 English sentences that keep the article's own framing.
	Summary string `json:"summary"`
	// SummaryZh and DevelopmentZh are the same two fields in Traditional Chinese
	// (Taiwan usage), written from the article itself under the same rules.
	SummaryZh     string `json:"summary_zh"`
	DevelopmentZh string `json:"development_zh"`
	// Development is the one new development the article reports, as a short
	// neutral English line; empty for explainers and side stories.
	Development string `json:"development"`
	// HappenedOn is YYYY-MM-DD, or empty when the article doesn't make it clear.
	HappenedOn        string   `json:"happened_on"`
	DateIsApproximate bool     `json:"date_is_approximate"`
	Entities          []Entity `json:"entities"`
	// Recaps are earlier developments the article retells, one English line each.
	Recaps []string `json:"recaps"`
}

type LinkInput struct {
	Headline    string
	Summary     string
	Development string
	Recaps      []string
	Entities    []string
	Candidates  []CandidateEvent
}

type CandidateEvent struct {
	ID    int64
	Title string
	Steps []CandidateStep
}

type CandidateStep struct {
	ArticleID   int64
	HappenedOn  string
	Development string
}

type LinkDecision struct {
	// EventID is the chosen candidate, or 0 for a new event.
	EventID int64 `json:"event_id"`
	// StepArticleID is the existing timeline step that already covers the
	// article's development, or 0 when the development is new to the event.
	StepArticleID int64 `json:"step_article_id"`
	// Confidence is 0–1; low values are surfaced in the admin tool.
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}

type TitleInput struct {
	Headline    string
	Summary     string
	Development string
}

// TranslateInput is English text to render in Traditional Chinese. Empty
// fields are skipped and come back empty.
type TranslateInput struct {
	Title       string
	Summary     string
	Development string
	// Names are the people, organizations and places the text refers to, in the
	// original script. The translation must use these forms, not its own guess.
	Names []string
}

type Translation struct {
	Title       string `json:"title"`
	Summary     string `json:"summary"`
	Development string `json:"development"`
}
