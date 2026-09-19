package ai

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
)

// Prompts are files so that a prompt change is a reviewable diff.
var (
	//go:embed prompts/summarize.md
	summarizePrompt string
	//go:embed prompts/summarize_event.md
	summarizeEventPrompt string
	//go:embed prompts/link.md
	linkPrompt string
	//go:embed prompts/title.md
	titlePrompt string
)

// maxBodyRunes bounds what is sent to the model. News articles are far
// shorter; this only guards against a bad extraction swallowing a whole page.
const maxBodyRunes = 12000

// OpenAI talks to any OpenAI-compatible chat-completions endpoint, such as
// api.openai.com or a LiteLLM proxy.
type OpenAI struct {
	client openai.Client
	model  string
}

func NewOpenAI(apiKey, baseURL, model string) *OpenAI {
	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithRequestTimeout(3 * time.Minute),
		option.WithMaxRetries(2),
	}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	return &OpenAI{client: openai.NewClient(opts...), model: model}
}

func (o *OpenAI) Model() string { return o.model }

func (o *OpenAI) SummarizeArticle(ctx context.Context, in ArticleInput) (ArticleAnalysis, error) {
	body := []rune(in.Body)
	if len(body) > maxBodyRunes {
		body = body[:maxBodyRunes]
	}
	user := fmt.Sprintf("Outlet: %s\nPublished: %s\nHeadline: %s\n\n%s", in.Outlet, in.PublishedOn, in.Headline, string(body))

	var out ArticleAnalysis
	if err := o.structured(ctx, summarizePrompt, user, "article_analysis", analysisSchema, &out); err != nil {
		return ArticleAnalysis{}, err
	}
	out.Summary = strings.TrimSpace(out.Summary)
	out.Development = strings.TrimSpace(out.Development)
	if out.Summary == "" {
		return ArticleAnalysis{}, errors.New("model returned an empty summary")
	}
	if out.HappenedOn != "" {
		if _, err := time.Parse(time.DateOnly, out.HappenedOn); err != nil {
			// A malformed date is treated as unknown rather than failing the article.
			out.HappenedOn, out.DateIsApproximate = "", false
		}
	}
	return out, nil
}

func (o *OpenAI) SummarizeEvent(ctx context.Context, in EventInput) (string, error) {
	const maxRunes = 24000
	// Bound unusually large, long-running events while retaining the title and
	// latest coverage. Re-marshal after trimming so the payload stays valid JSON.
	payloadFor := func() ([]byte, error) {
		return json.Marshal(map[string]any{
			"title": in.Title, "timeline": in.Timeline, "article_summaries": in.Summaries,
		})
	}
	payload, err := payloadFor()
	for err == nil && len([]rune(string(payload))) > maxRunes && len(in.Summaries) > 1 {
		in.Summaries = in.Summaries[1:]
		payload, err = payloadFor()
	}
	for err == nil && len([]rune(string(payload))) > maxRunes && len(in.Timeline) > 1 {
		in.Timeline = in.Timeline[1:]
		payload, err = payloadFor()
	}
	if err != nil {
		return "", fmt.Errorf("summarize event payload: %w", err)
	}
	resp, err := o.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: o.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(summarizeEventPrompt),
			openai.UserMessage(string(payload)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("summarize event: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", errors.New("summarize event: no choices returned")
	}
	summary := strings.TrimSpace(resp.Choices[0].Message.Content)
	if summary == "" {
		return "", errors.New("summarize event: empty summary")
	}
	return summary, nil
}

func (o *OpenAI) LinkToEvent(ctx context.Context, in LinkInput) (LinkDecision, error) {
	if len(in.Candidates) == 0 {
		return LinkDecision{Confidence: 1, Reason: "no candidate events"}, nil
	}
	payload, err := json.MarshalIndent(linkPayload(in), "", "  ")
	if err != nil {
		return LinkDecision{}, err
	}

	var out LinkDecision
	if err := o.structured(ctx, linkPrompt, string(payload), "link_decision", linkSchema, &out); err != nil {
		return LinkDecision{}, err
	}
	return sanitizeDecision(out, in.Candidates), nil
}

func (o *OpenAI) TitleEvent(ctx context.Context, in TitleInput) (string, error) {
	user := fmt.Sprintf("Headline: %s\nSummary: %s\nDevelopment: %s", in.Headline, in.Summary, in.Development)
	resp, err := o.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: o.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(titlePrompt),
			openai.UserMessage(user),
		},
	})
	if err != nil {
		return "", fmt.Errorf("title event: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", errors.New("title event: no choices returned")
	}
	title := strings.Trim(strings.TrimSpace(resp.Choices[0].Message.Content), `"“”.`)
	if title == "" {
		return "", errors.New("title event: empty title")
	}
	return title, nil
}

// structured runs one chat completion constrained to a strict JSON schema.
func (o *OpenAI) structured(ctx context.Context, system, user, name string, schema map[string]any, out any) error {
	resp, err := o.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: o.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(system),
			openai.UserMessage(user),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &shared.ResponseFormatJSONSchemaParam{
				JSONSchema: shared.ResponseFormatJSONSchemaJSONSchemaParam{
					Name:   name,
					Strict: openai.Bool(true),
					Schema: schema,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if len(resp.Choices) == 0 {
		return fmt.Errorf("%s: no choices returned", name)
	}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), out); err != nil {
		return fmt.Errorf("%s: decode model output: %w", name, err)
	}
	return nil
}

// sanitizeDecision drops ids the model invented: only given candidates and
// their steps are valid answers.
func sanitizeDecision(d LinkDecision, candidates []CandidateEvent) LinkDecision {
	d.Confidence = min(max(d.Confidence, 0), 1)
	for _, c := range candidates {
		if c.ID != d.EventID {
			continue
		}
		for _, s := range c.Steps {
			if s.ArticleID == d.StepArticleID {
				return d
			}
		}
		d.StepArticleID = 0
		return d
	}
	d.EventID, d.StepArticleID = 0, 0
	return d
}

func linkPayload(in LinkInput) map[string]any {
	candidates := make([]map[string]any, 0, len(in.Candidates))
	for _, c := range in.Candidates {
		steps := make([]map[string]any, 0, len(c.Steps))
		for _, s := range c.Steps {
			steps = append(steps, map[string]any{"article_id": s.ArticleID, "date": s.HappenedOn, "development": s.Development})
		}
		candidates = append(candidates, map[string]any{"event_id": c.ID, "title": c.Title, "timeline": steps})
	}
	return map[string]any{
		"article": map[string]any{
			"headline":    in.Headline,
			"summary":     in.Summary,
			"development": in.Development,
			"recaps":      in.Recaps,
			"entities":    in.Entities,
		},
		"candidates": candidates,
	}
}

// Strict structured outputs require every property to be listed as required
// and additionalProperties to be false, at every level.
var analysisSchema = map[string]any{
	"type":                 "object",
	"additionalProperties": false,
	"required":             []string{"summary", "development", "happened_on", "date_is_approximate", "entities", "recaps"},
	"properties": map[string]any{
		"summary":             map[string]any{"type": "string"},
		"development":         map[string]any{"type": "string"},
		"happened_on":         map[string]any{"type": "string", "description": "YYYY-MM-DD, or an empty string when unclear"},
		"date_is_approximate": map[string]any{"type": "boolean"},
		"entities": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"name", "kind", "aliases"},
				"properties": map[string]any{
					"name":    map[string]any{"type": "string"},
					"kind":    map[string]any{"type": "string", "enum": []string{"person", "organization", "place"}},
					"aliases": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				},
			},
		},
		"recaps": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
	},
}

var linkSchema = map[string]any{
	"type":                 "object",
	"additionalProperties": false,
	"required":             []string{"event_id", "step_article_id", "confidence", "reason"},
	"properties": map[string]any{
		"event_id":        map[string]any{"type": "integer"},
		"step_article_id": map[string]any{"type": "integer"},
		"confidence":      map[string]any{"type": "number"},
		"reason":          map[string]any{"type": "string"},
	},
}
