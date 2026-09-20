package httpapi

import (
	"sort"
	"time"

	"github.com/yorukot/ground-news-tw/internal/db"
)

// taipei is the zone used to turn a publish time into a calendar date when
// an article gives no explicit date for its development.
var taipei = mustLoadLocation("Asia/Taipei")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		// Fixed offset fallback; Taiwan has no daylight saving time.
		return time.FixedZone(name, 8*60*60)
	}
	return loc
}

// buildEventBody turns an event's article rows into the coverage list (wire
// reprints folded under their original) and the timeline (oldest step first).
func buildEventBody(rows []db.ListEventArticlesRow, l lang) ([]Article, []TimelineStep) {
	inEvent := make(map[int64]bool, len(rows))
	for _, r := range rows {
		inEvent[r.ID] = true
	}

	// cardID maps every article to the card that displays it: itself, or the
	// original when it is a reprint whose original is in this event.
	cardID := func(r db.ListEventArticlesRow) int64 {
		if r.ReprintOfID.Valid && inEvent[r.ReprintOfID.Int64] {
			return r.ReprintOfID.Int64
		}
		return r.ID
	}

	reprints := make(map[int64][]Reprint)
	for _, r := range rows {
		if id := cardID(r); id != r.ID {
			reprints[id] = append(reprints[id], Reprint{
				Outlet:      outletOf(r),
				URL:         r.Url,
				PublishedAt: r.PublishedAt,
			})
		}
	}

	articles := make([]Article, 0, len(rows))
	for _, r := range rows { // rows arrive newest first
		if cardID(r) != r.ID {
			continue
		}
		rp := reprints[r.ID]
		sort.SliceStable(rp, func(i, j int) bool { return rp[i].PublishedAt.Before(rp[j].PublishedAt) })
		if rp == nil {
			rp = []Reprint{}
		}
		articles = append(articles, Article{
			ID:           r.ID,
			Outlet:       outletOf(r),
			Headline:     r.Headline,
			URL:          r.Url,
			ImageURL:     r.ImageUrl,
			PublishedAt:  r.PublishedAt,
			Summary:      l.text(r.Summary, r.SummaryZh),
			HeadlineOnly: r.HeadlineOnly,
			Reprints:     rp,
		})
	}

	// Which step each card reports, so reprints inherit their original's step.
	stepOfCard := make(map[int64]int64)
	for _, r := range rows {
		if r.StepArticleID.Valid {
			stepOfCard[r.ID] = r.StepArticleID.Int64
		}
	}

	steps := make(map[int64]*TimelineStep)
	for _, r := range rows {
		if !r.StepArticleID.Valid || r.StepArticleID.Int64 != r.ID || r.Development == "" {
			continue
		}
		step := &TimelineStep{
			ArticleID:         r.ID,
			Development:       l.text(r.Development, r.DevelopmentZh),
			DateIsApproximate: r.DateIsApproximate,
			Reports:           []StepReport{},
		}
		if r.HappenedOn.Valid {
			step.HappenedOn = r.HappenedOn.Time.Format(time.DateOnly)
		} else {
			// Plan: an unclear date uses the first report date, marked approximate.
			step.HappenedOn = r.PublishedAt.In(taipei).Format(time.DateOnly)
			if r.PublishedAt.IsZero() {
				step.HappenedOn = r.FirstSeenAt.In(taipei).Format(time.DateOnly)
			}
			step.DateIsApproximate = true
		}
		steps[r.ID] = step
	}

	for _, r := range rows {
		stepID, ok := stepOfCard[cardID(r)]
		if !ok {
			continue
		}
		step, ok := steps[stepID]
		if !ok {
			continue
		}
		step.Reports = append(step.Reports, StepReport{
			ArticleID:   cardID(r),
			Outlet:      outletOf(r),
			PublishedAt: r.PublishedAt,
		})
	}

	timeline := make([]TimelineStep, 0, len(steps))
	for _, s := range steps {
		sort.SliceStable(s.Reports, func(i, j int) bool { return s.Reports[i].PublishedAt.Before(s.Reports[j].PublishedAt) })
		// An outlet that ran several articles on one development is listed
		// once, by its earliest article.
		seen := make(map[int64]bool, len(s.Reports))
		unique := s.Reports[:0]
		for _, r := range s.Reports {
			if !seen[r.Outlet.ID] {
				seen[r.Outlet.ID] = true
				unique = append(unique, r)
			}
		}
		s.Reports = unique
		timeline = append(timeline, *s)
	}
	sort.SliceStable(timeline, func(i, j int) bool {
		if timeline[i].HappenedOn != timeline[j].HappenedOn {
			return timeline[i].HappenedOn < timeline[j].HappenedOn
		}
		return firstReport(timeline[i]).Before(firstReport(timeline[j]))
	})
	return articles, timeline
}

func firstReport(s TimelineStep) time.Time {
	if len(s.Reports) == 0 {
		return time.Time{}
	}
	return s.Reports[0].PublishedAt
}

func outletOf(r db.ListEventArticlesRow) Outlet {
	return Outlet{ID: r.OutletID, Slug: r.OutletSlug, Name: r.OutletName}
}
