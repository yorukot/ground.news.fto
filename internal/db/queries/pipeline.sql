-- Queries used by the ingestion pipeline (internal/jobs, internal/link).

-- name: ExistingArticleURLs :many
SELECT url FROM articles WHERE url = ANY(sqlc.arg(urls)::text[]);

-- name: InsertFetchedArticle :one
-- Returns no row when the URL is already stored, which makes a retried fetch a no-op.
INSERT INTO articles (outlet_id, url, headline, image_url, body, published_at, content_hash, minhash)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (url) DO UPDATE SET headline=EXCLUDED.headline,body=EXCLUDED.body,
image_url=CASE WHEN articles.image_url='' THEN EXCLUDED.image_url ELSE articles.image_url END,
published_at=EXCLUDED.published_at,content_hash=EXCLUDED.content_hash,minhash=EXCLUDED.minhash
WHERE articles.body=''
RETURNING id;

-- name: GetArticleForDedupe :one
SELECT id, outlet_id, minhash, published_at, reprint_of_id, manual_link
FROM articles
WHERE id = $1;

-- name: ListRecentSignatures :many
-- Wire stories are reprinted within hours, so only recent originals are compared.
SELECT a.id, a.minhash, a.event_id
FROM articles a
WHERE a.first_seen_at >= sqlc.arg(since)
  AND a.id < sqlc.arg(exclude_id)
  AND a.reprint_of_id IS NULL
  AND cardinality(a.minhash) > 0
  AND NOT EXISTS(SELECT 1 FROM crawl_urls c WHERE c.article_id=a.id AND NOT c.eligible)
ORDER BY a.published_at, a.id;

-- name: SetArticleReprint :exec
UPDATE articles
SET reprint_of_id = $2, event_id = $3
WHERE id = $1;

-- name: GetArticleForAnalysis :one
SELECT a.id, a.headline, a.body, a.published_at, a.reprint_of_id, o.name AS outlet_name
FROM articles a
JOIN outlets o ON o.id = a.outlet_id
WHERE a.id = $1;

-- name: GetSummaryVersion :one
SELECT model, prompt_version FROM summaries WHERE article_id = $1;

-- name: SetArticleAnalysis :exec
UPDATE articles
SET development = $2, happened_on = $3, date_is_approximate = $4
WHERE id = $1;

-- name: GetArticleForLink :one
SELECT
    a.id, a.headline, a.development, a.published_at, a.first_seen_at, a.manual_link, a.step_article_id, a.event_id, a.reprint_of_id,
    COALESCE(s.text, '')::text AS summary,
    COALESCE(s.recaps, '{}')::text[] AS recaps
FROM articles a
LEFT JOIN summaries s ON s.article_id = a.id
WHERE a.id = $1;

-- name: SetArticleLink :exec
UPDATE articles
SET event_id = $2, step_article_id = $3, link_confidence = $4
WHERE id = $1;

-- name: SetReprintsEvent :exec
-- Reprints always live in their original's event.
UPDATE articles
SET event_id = $2
WHERE reprint_of_id = $1;

-- name: ListEventSteps :many
-- The timeline steps of the given events, oldest first.
SELECT a.event_id, a.id, a.development, a.happened_on, a.published_at
FROM articles a
WHERE a.event_id = ANY(sqlc.arg(event_ids)::bigint[])
  AND a.step_article_id = a.id
  AND a.development <> ''
ORDER BY a.event_id, COALESCE(a.happened_on, a.published_at::date), a.published_at;

-- name: GetEventTitles :many
SELECT id, title FROM events WHERE id = ANY(sqlc.arg(event_ids)::bigint[]);

-- name: GetEventForSummary :one
SELECT id, title, updated_at, summary_model, summary_prompt_version
FROM events
WHERE id = $1;

-- name: ListEventsNeedingSummary :many
SELECT e.id
FROM events e
WHERE (e.summary_model <> sqlc.arg(model) OR e.summary_prompt_version <> sqlc.arg(prompt_version))
  AND EXISTS (
      SELECT 1 FROM articles a JOIN summaries s ON s.article_id = a.id
      WHERE a.event_id = e.id AND a.reprint_of_id IS NULL
      AND a.url NOT LIKE 'https://example.com/ground-sample/%'
  )
ORDER BY e.updated_at DESC
LIMIT sqlc.arg(max_results);

-- name: ListEventSummarySources :many
SELECT a.development, s.text AS summary
FROM articles a
JOIN summaries s ON s.article_id = a.id
WHERE a.event_id = sqlc.arg(event_id)::bigint AND a.reprint_of_id IS NULL
ORDER BY a.published_at ASC, a.id ASC;

-- name: SetEventSummary :execrows
-- Do not let a slower job overwrite an overview made from newer coverage.
UPDATE events
SET summary = sqlc.arg(summary),
    summary_model = sqlc.arg(model),
    summary_prompt_version = sqlc.arg(prompt_version)
WHERE id = sqlc.arg(id) AND updated_at = sqlc.arg(expected_updated_at);

-- name: TouchEvent :exec
-- Moves the event to the top of the homepage and refreshes its search text.
UPDATE events
SET updated_at = GREATEST(updated_at, sqlc.arg(updated_at)),
    timeline_text = sqlc.arg(timeline_text),
    summary_prompt_version = ''
WHERE id = sqlc.arg(id);

-- name: FindEntity :one
-- An exact match on the canonical name or any alias, within one kind.
SELECT id, canonical_name, aliases
FROM entities
WHERE kind = sqlc.arg(kind)
  AND (canonical_name = sqlc.arg(name) OR sqlc.arg(name)::text = ANY(aliases))
ORDER BY id
LIMIT 1;

-- name: CreateEntity :one
INSERT INTO entities (canonical_name, kind, aliases, search_text)
VALUES ($1, $2, $3, $4)
ON CONFLICT (canonical_name) DO UPDATE SET canonical_name = EXCLUDED.canonical_name
RETURNING id;

-- name: SetEntityAliases :exec
UPDATE entities SET aliases = $2, search_text = $3 WHERE id = $1;

-- name: LinkArticleEntity :exec
INSERT INTO article_entities (article_id, entity_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: LinkEventEntity :exec
INSERT INTO event_entities (event_id, entity_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: ListArticleEntities :many
SELECT e.id, e.canonical_name
FROM article_entities ae
JOIN entities e ON e.id = ae.entity_id
WHERE ae.article_id = $1
ORDER BY e.id;
-- name: ListArticlesMissingImage :many
WITH ranked AS (
    SELECT
        a.id,
        a.url,
        a.event_id,
        e.updated_at AS event_updated_at,
        a.published_at,
        row_number() OVER (PARTITION BY a.event_id ORDER BY a.published_at DESC, a.id DESC) AS event_rank,
        EXISTS (
            SELECT 1 FROM articles image
            WHERE image.event_id = a.event_id AND image.image_url <> ''
        ) AS event_has_image
    FROM articles a
    LEFT JOIN events e ON e.id = a.event_id
    WHERE a.image_url = ''
)
SELECT id, url
FROM ranked
ORDER BY
    (event_id IS NOT NULL AND NOT event_has_image AND event_rank = 1) DESC,
    (event_id IS NOT NULL) DESC,
    event_updated_at DESC NULLS LAST,
    published_at DESC,
    id DESC
LIMIT sqlc.arg(max_results);

-- name: SetArticleImage :exec
UPDATE articles SET image_url = $2 WHERE id = $1 AND image_url = '';
