# Ingestion validation — 2026-09-19

## Automated checks

- Backend tests, including PostgreSQL integration tests, pass with the race detector.
- Concurrent AI dispatch respects the shared daily quota, balances outlets,
  survives another dispatch, resets at Taiwan midnight, and pauses without AI.
- Discovery persists more than the former forty-URL limit, merges metadata,
  resumes pagination, retains successful pages after a subsequent failure,
  and rejects unexpected pagination hosts.
- Fetching upgrades an existing empty-body article without changing its ID.
  AI-use declarations do not suppress its body; redirect target robots rules
  still apply. Missing publication dates remain unknown.
- Extraction fixtures cover eight outlet selectors, PTS's separate introduction,
  LTN's array/object JSON responses, and fallback from short CSS to JSON-LD.
- Reprint candidates only point to lower article IDs, preventing reference cycles.
- Frontend typecheck, nineteen tests, Biome checks, and production build pass.

## Live source probes

Ten articles were attempted for each of the eighteen enabled outlets: CNA, PTS,
UDN, LTN, China Times, ETtoday, SETN, TVBS, EBC, FTV, CTi, Mirror Media, Storm,
Up Media, Newtalk, NOWnews, The News Lens, and The Reporter.

Of 180 attempts, 179 passed extraction heuristics. One ETtoday request failed
with an HTTP/2 compression error. This is a retrieval/extraction smoke result,
**not a 99% completeness or accuracy claim**. Opening and ending excerpts were
inspected for selected results; the full set has not been independently audited.

The probes exposed and led to fixes for UDN footer contamination, China Times
related links, LTN advertisements, PTS's omitted introduction, and selected
inline promotional tails. LTN's first-page array format was additionally
verified against the live endpoint after fixing its parser; that follow-up
discovered 210 URLs without source errors. Some captions and embedded related
material may remain, especially in generic Readability output.

## Local rollout and limits

The local database was backed up before applying migrations 00004 and 00005.
Changed outlet settings were synchronized and seven-day backfill jobs enqueued.
A real-model worker smoke run used a temporary twenty-new-article daily quota.
All twenty article summaries and event links were successfully persisted. Restarting
the worker preserved its twenty admission reservations. No production deployment
was performed. The smoke worker was stopped after verification; its durable
backfill and fetch queues can be resumed with `make worker`. This loads `.env`
and uses the 500-article default unless the environment overrides it.

This does not validate seven complete archive days for every outlet: RSS and
non-paginated lists expose only what publishers currently provide. Large sitemap
walks and fetch queues require more time. Sitemap modification dates are not
treated as article publication dates. Full-day quota behavior, long-running
queue growth, and cross-outlet event matching quality still need operational
observation. The checked-in default remains 500 new articles per Taiwan day;
that is not a cap on API calls or money.

See [operating instructions](outlets.md) for status, synchronization, backfill,
and repeatable source checks.
