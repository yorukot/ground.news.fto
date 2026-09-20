# News ingestion

All 18 enabled outlets attempt full-text extraction. AI-use declarations no
longer select a headline-only mode. The fetcher still checks the applicable
robots paths, identifies itself, waits three seconds per host (including
redirects), and handles 429 Retry-After. Commonwealth and Yahoo remain disabled.

## Pipeline and limits

Every five minutes, discovery reads configured feeds, sitemaps and lists.
Discovered URLs are persisted before processing: the former 40-article cap is
gone. A source cursor is committed with its URLs, and successful pages survive
a later page failure. Live discovery processes five list pages or ten sitemap
pages per batch; backfill processes twenty list pages. Sources with no next page
finish immediately; this is not a claim that the publisher's entire archive was
covered. Source errors remain visible in the report.

The fetch dispatcher admits up to 100 URLs per minute; River runs four fetch
workers. Failed requests retry through River, while non-retryable failures are
recorded separately. A completed URL is not routinely re-fetched. Existing
missing-body articles are upgraded in place, preserving their IDs.

Extraction tries configured CSS selectors, JSON-LD articleBody, then
Readability. A short or invalid selector result falls through instead of
blocking extraction. PTS has an explicit lead selector because its introduction
is outside the paragraph tags. LTN's paginated list uses a small dedicated
decoder for its public JSON endpoint. There is no browser service.

Explicit public-interest categories remain eligible; entertainment, sports,
shopping, fashion and travel are excluded when the source identifies them.
Unknown categories remain eligible. This deliberately conservative filter is
not semantic topic classification, and mixed-topic material will still arrive.

Publication times that cannot be recovered are stored as Go's zero-time
sentinel on the legacy article schema and returned as null in the public API.
The URL ledger stores SQL NULL. Discovery time is not presented as publication
time. An approximate timeline date can use first-seen time.

## AI budget

`AI_MAX_NEW_ARTICLES_PER_DAY=500` caps newly admitted articles per Asia/Taipei
calendar day. Dispatch runs every five minutes, at most twenty per run, balancing
daily admissions among outlets and selecting the oldest waiting article within
each outlet. Unused slots can go to other outlets. The database serializes
admission reservations and River enqueue in one transaction. Restarts and
retries do not reset the quota. Completed reprints and existing summaries do
not consume a new article slot.

This is an article quota, not a currency or API-call cap: linking, titles,
event summaries and retries can make additional calls. AI concurrency remains
two; linking remains serialized. Waiting articles keep their bodies. Monitor
the oldest waiting time to see when the quota is insufficient.

Without OPENAI_API_KEY, AI work pauses while crawling continues. Fake AI is
available only with `ALLOW_FAKE_AI=true`; never use it for production summaries.

## Commands

These commands read configuration from the environment. Unlike `make`, a direct
`go run` does not automatically load `.env`.

```sh
go run ./cmd/app crawl-check udn --count 10 --json
go run ./cmd/app crawl-sync --outlet udn
go run ./cmd/app crawl-backfill --days 7
go run ./cmd/app crawl-status --json
```

- `crawl-check`: inspect checked-in settings without DB writes or AI calls;
  reports source errors plus each attempted article's title, time, extractor,
  length, opening and ending. Success means extraction passed its heuristic,
  not an independent guarantee of completeness. Read the excerpts.
- `crawl-sync`: explicitly update one registered outlet's crawl configuration;
  preserve its enabled setting. Do not use seeding to overwrite tuning.
- `crawl-backfill`: enqueue resumable source discovery and missing-body recovery.
  Re-running the same outlet/date window resumes its cursor. It cannot retrieve
  archive pages that the source no longer exposes. Jobs require a running worker.
- `crawl-status`: per-outlet discovery, body retrieval, failures, waiting analysis,
  oldest wait and last success, plus per-source cursors and errors.

## Upgrade

Stop the old worker before migration; the old code depends on the removed
coverage column. Back up the database, run `app migrate`, sync changed outlet
settings, then start the new API, web and worker. Backfill seven days.

Migration removes automatic legacy headline event associations while preserving
identified manual links (the old admin wrote confidence 1). Existing sample
URLs are never crawled. The old `match_headline` job kind remains registered as
a no-op only to drain already queued jobs. The public `headlineOnly` boolean
remains compatible, but now means this article lacks a retrieved body; its UI
message no longer describes a publisher policy.

Inspect reports after startup and after a full quota day. Do not equate a missing
article with a publisher choosing not to report an event. Fetch coverage,
extraction quality, and AI processing completion are separate measurements.
