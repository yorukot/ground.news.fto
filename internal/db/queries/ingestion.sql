-- name: DiscoverURL :exec
INSERT INTO crawl_urls(url,outlet_id,source,headline,category,scope_reason,eligible,published_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT(url) DO UPDATE SET
 headline=CASE WHEN crawl_urls.headline='' THEN EXCLUDED.headline ELSE crawl_urls.headline END,
 category=CASE WHEN crawl_urls.category='' THEN EXCLUDED.category ELSE crawl_urls.category END,
 eligible=CASE WHEN crawl_urls.category='' THEN EXCLUDED.eligible ELSE crawl_urls.eligible END,
 scope_reason=CASE WHEN crawl_urls.category='' THEN EXCLUDED.scope_reason ELSE crawl_urls.scope_reason END,
 published_at=COALESCE(crawl_urls.published_at,EXCLUDED.published_at),last_seen_at=now();

-- name: GetCrawlURL :one
SELECT * FROM crawl_urls WHERE url=$1;

-- name: ListFetchCandidates :many
WITH candidates AS (
SELECT url,row_number() OVER(PARTITION BY outlet_id ORDER BY (article_id IS NOT NULL) DESC,first_seen_at,url) AS position FROM crawl_urls
WHERE result IN ('pending','retry') AND next_attempt_at<=now() AND eligible
AND outlet_id IN (SELECT id FROM outlets WHERE enabled)
)
SELECT c.url,c.outlet_id,c.published_at FROM crawl_urls c JOIN candidates d ON d.url=c.url
ORDER BY d.position,c.first_seen_at,c.url LIMIT $1 FOR UPDATE OF c SKIP LOCKED;

-- name: ReserveFetch :exec
UPDATE crawl_urls SET next_attempt_at=now()+interval '30 minutes' WHERE url=$1;

-- name: RecordFetchFailure :exec
UPDATE crawl_urls SET result=$2,http_status=$3,error=$4,next_attempt_at=$5 WHERE url=$1;

-- name: RecordFetchSuccess :exec
UPDATE crawl_urls SET result='success',http_status=200,error='',article_id=$2,
 extractor=$3,body_length=$4,fetched_at=now(),category=$5,eligible=$6,scope_reason=$7,
 published_at=COALESCE(sqlc.narg(published_at)::timestamptz,published_at)
WHERE url=$1;

-- name: GetSourceProgress :one
SELECT * FROM crawl_sources WHERE outlet_id=$1 AND source=$2 AND mode=$3;

-- name: SaveSourceProgress :exec
INSERT INTO crawl_sources(outlet_id,source,mode,since_at,progress,error)
VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(outlet_id,source,mode)
DO UPDATE SET since_at=EXCLUDED.since_at,progress=EXCLUDED.progress,error=EXCLUDED.error,updated_at=now();

-- name: MarkAnalysisReady :exec
INSERT INTO analysis_pending(article_id) VALUES($1) ON CONFLICT DO NOTHING;

-- name: CountDailyAdmissions :one
SELECT count(*) FROM ai_admissions WHERE admitted_on=$1;

-- name: NextAnalysisCandidate :one
SELECT a.id,a.outlet_id FROM analysis_pending p JOIN articles a ON a.id=p.article_id
JOIN outlets o ON o.id=a.outlet_id
WHERE o.enabled AND a.body<>'' AND a.reprint_of_id IS NULL
AND NOT EXISTS(SELECT 1 FROM summaries s WHERE s.article_id=a.id)
AND NOT EXISTS(SELECT 1 FROM ai_admissions d WHERE d.article_id=a.id)
AND NOT EXISTS(SELECT 1 FROM crawl_urls c WHERE c.article_id=a.id AND NOT c.eligible)
ORDER BY
 (SELECT count(*) FROM ai_admissions d WHERE d.outlet_id=a.outlet_id AND d.admitted_on=$1),
 (SELECT max(d.admitted_at) FROM ai_admissions d WHERE d.outlet_id=a.outlet_id) ASC NULLS FIRST,
 p.ready_at,a.id LIMIT 1;

-- name: AdmitAnalysis :exec
INSERT INTO ai_admissions(article_id,outlet_id,admitted_on) VALUES($1,$2,$3);

-- name: HasAdmission :one
SELECT EXISTS(SELECT 1 FROM ai_admissions WHERE article_id=$1);

-- name: CrawlStatus :many
SELECT o.slug,
 count(c.url)::bigint AS discovered,
 count(c.url) FILTER(WHERE c.result='success')::bigint AS fetched,
 count(c.url) FILTER(WHERE c.result NOT IN ('pending','success'))::bigint AS failed,
 count(c.url) FILTER(WHERE c.result='success' AND c.eligible AND a.reprint_of_id IS NULL AND s.article_id IS NULL)::bigint AS awaiting_analysis,
 min(c.first_seen_at) FILTER(WHERE c.result='success' AND c.eligible AND a.reprint_of_id IS NULL AND s.article_id IS NULL) AS oldest_waiting,
 max(c.fetched_at) AS last_success
FROM outlets o LEFT JOIN crawl_urls c ON c.outlet_id=o.id
LEFT JOIN articles a ON a.id=c.article_id LEFT JOIN summaries s ON s.article_id=a.id
WHERE sqlc.arg(slug)::text='' OR o.slug=sqlc.arg(slug)
GROUP BY o.id,o.slug ORDER BY o.slug;

-- name: CrawlFailures :many
SELECT o.slug,c.result,c.http_status,c.error,count(*)::bigint AS articles
FROM crawl_urls c JOIN outlets o ON o.id=c.outlet_id
WHERE c.error<>'' AND (sqlc.arg(slug)::text='' OR o.slug=sqlc.arg(slug))
GROUP BY o.slug,c.result,c.http_status,c.error ORDER BY o.slug,articles DESC;

-- name: SourceStatus :many
SELECT o.slug,s.source,s.mode,s.progress,s.error,s.updated_at FROM crawl_sources s
JOIN outlets o ON o.id=s.outlet_id
WHERE sqlc.arg(slug)::text='' OR o.slug=sqlc.arg(slug) ORDER BY o.slug,s.source,s.mode;
