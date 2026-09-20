package ai

import (
	"context"
	"strings"
	"unicode/utf8"
)

// Fake is a deterministic Client for tests and explicit ALLOW_FAKE_AI development.
// It
// makes no attempt at quality: the summary is the article's opening, and an
// article joins a candidate event only when a recap or its development
// literally matches one of that event's steps.
type Fake struct{}

func (Fake) Model() string { return "fake" }

func (Fake) SummarizeArticle(_ context.Context, in ArticleInput) (ArticleAnalysis, error) {
	summary := firstRunes(strings.Join(strings.Fields(in.Body), " "), 200)
	// The fake works on the Chinese source text, so both languages are the same.
	return ArticleAnalysis{
		Summary:       summary,
		SummaryZh:     summary,
		Development:   in.Headline,
		DevelopmentZh: in.Headline,
		HappenedOn:    in.PublishedOn,
		Entities:      []Entity{},
		Recaps:        []string{},
	}, nil
}

func (Fake) SummarizeEvent(_ context.Context, in EventInput) (string, error) {
	if len(in.Summaries) > 0 {
		return firstRunes(strings.Join(in.Summaries, " "), 400), nil
	}
	return firstRunes(strings.Join(in.Timeline, ". "), 400), nil
}

func (Fake) LinkToEvent(_ context.Context, in LinkInput) (LinkDecision, error) {
	for _, c := range in.Candidates {
		for _, s := range c.Steps {
			if s.Development == in.Development {
				return LinkDecision{EventID: c.ID, StepArticleID: s.ArticleID, Confidence: 1, Reason: "same development"}, nil
			}
			for _, recap := range in.Recaps {
				if recap == s.Development {
					return LinkDecision{EventID: c.ID, Confidence: 0.9, Reason: "recap matches a step"}, nil
				}
			}
		}
	}
	return LinkDecision{Confidence: 0.5, Reason: "no literal match"}, nil
}

func (Fake) TitleEvent(_ context.Context, in TitleInput) (string, error) {
	return firstRunes(in.Headline, 80), nil
}

func (Fake) Translate(_ context.Context, in TranslateInput) (Translation, error) {
	return Translation{Title: in.Title, Summary: in.Summary, Development: in.Development}, nil
}

func firstRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}
