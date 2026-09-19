// Package jobs is the ingestion pipeline as River jobs:
//
//	crawl_all → crawl_outlet → fetch_article → dedupe_article → summarize_article → link_article → summarize_event
//
// Outlets in headline-only coverage take a shorter path that never fetches an
// article page and never calls the model:
//
//	crawl_all → crawl_outlet → match_headline
//
// Each job writes its result and enqueues the next job in one transaction, so
// a crash can neither lose an article nor process it twice. Every worker is
// also idempotent, because River retries a job whose process died mid-run.
package jobs

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/yorukot/ground-news-tw/internal/ai"
	"github.com/yorukot/ground-news-tw/internal/crawl"
	"github.com/yorukot/ground-news-tw/internal/link"
)

const (
	QueueCrawl = "crawl"
	QueueFetch = "fetch"
	QueueAI    = "ai"
	// QueueLink has exactly one worker; see link.Linker.Link.
	QueueLink = "link"

	crawlInterval        = 5 * time.Minute
	eventSummaryInterval = 10 * time.Minute
	// maxArticleAge skips old items that feeds sometimes resurface.
	maxArticleAge = 7 * 24 * time.Hour
	// reprintWindow is how far back wire originals are looked for.
	reprintWindow = 72 * time.Hour
)

type Deps struct {
	Pool    *pgxpool.Pool
	AI      ai.Client
	Fetcher *crawl.Fetcher
	// MaxNewPerCrawl bounds how many articles one crawl of one outlet can
	// enqueue, so a first run, or a feed that suddenly lists its archive,
	// can't flood an outlet's site or the model budget.
	MaxNewPerCrawl int
}

func Queues() map[string]river.QueueConfig {
	return map[string]river.QueueConfig{
		river.QueueDefault: {MaxWorkers: 4},
		QueueCrawl:         {MaxWorkers: 2},
		QueueFetch:         {MaxWorkers: 4},
		QueueAI:            {MaxWorkers: 2},
		QueueLink:          {MaxWorkers: 1},
	}
}

func Workers(d Deps) *river.Workers {
	workers := river.NewWorkers()
	river.AddWorker(workers, &CrawlAllWorker{deps: d})
	river.AddWorker(workers, &CrawlOutletWorker{deps: d})
	river.AddWorker(workers, &FetchArticleWorker{deps: d})
	river.AddWorker(workers, &DedupeArticleWorker{deps: d})
	river.AddWorker(workers, &SummarizeArticleWorker{deps: d})
	river.AddWorker(workers, &LinkArticleWorker{linker: &link.Linker{Pool: d.Pool, AI: d.AI}})
	river.AddWorker(workers, &SummarizeEventWorker{deps: d})
	river.AddWorker(workers, &EnqueueEventSummariesWorker{deps: d})
	river.AddWorker(workers, &MatchHeadlineWorker{deps: d})
	return workers
}

func PeriodicJobs() []*river.PeriodicJob {
	return []*river.PeriodicJob{
		river.NewPeriodicJob(
			river.PeriodicInterval(crawlInterval),
			func() (river.JobArgs, *river.InsertOpts) { return CrawlAllArgs{}, nil },
			&river.PeriodicJobOpts{RunOnStart: true},
		),
		river.NewPeriodicJob(
			river.PeriodicInterval(eventSummaryInterval),
			func() (river.JobArgs, *river.InsertOpts) { return EnqueueEventSummariesArgs{}, nil },
			&river.PeriodicJobOpts{RunOnStart: true},
		),
	}
}
