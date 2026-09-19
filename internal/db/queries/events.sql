-- name: ListEvents :many
-- Keyset pagination, most recently updated first. For the first page pass a
-- far-future cursor_updated_at and the maximum bigint as cursor_id.
SELECT
    e.id,
    e.title,
    e.first_seen_at,
    e.updated_at,
    (SELECT count(*) FROM articles a WHERE a.event_id = e.id)::bigint AS article_count,
    (SELECT count(DISTINCT a.outlet_id) FROM articles a WHERE a.event_id = e.id)::bigint AS outlet_count,
    COALESCE((
        SELECT a.development
        FROM articles a
        WHERE a.event_id = e.id AND a.step_article_id = a.id
        ORDER BY COALESCE(a.happened_on, a.published_at::date) DESC, a.published_at DESC
        LIMIT 1
    ), '')::text AS latest_development
FROM events e
WHERE (e.updated_at, e.id) < (sqlc.arg(cursor_updated_at)::timestamptz, sqlc.arg(cursor_id)::bigint)
  AND EXISTS (SELECT 1 FROM articles a WHERE a.event_id = e.id)
ORDER BY e.updated_at DESC, e.id DESC
LIMIT sqlc.arg(page_size);

-- name: GetEvent :one
SELECT
    e.id,
    e.title,
    e.first_seen_at,
    e.updated_at,
    (SELECT count(*) FROM articles a WHERE a.event_id = e.id)::bigint AS article_count,
    (SELECT count(DISTINCT a.outlet_id) FROM articles a WHERE a.event_id = e.id)::bigint AS outlet_count
FROM events e
WHERE e.id = $1;

-- name: CreateEvent :one
INSERT INTO events (title, first_seen_at, updated_at)
VALUES ($1, $2, $3)
RETURNING id;

-- name: SetEventTimelineText :exec
-- Called whenever an event gains a step, so the search index stays current.
UPDATE events
SET timeline_text = $2
WHERE id = $1;

-- name: SearchEventCandidates :many
-- Candidate events for a new article, by full-text match of the article's
-- development and recap lines against event titles and timelines. Terms are
-- OR-ed (plainto_tsquery would AND them, which is far too strict for a
-- paraphrased follow-up) and ranked, so a partial match still surfaces.
-- There is deliberately no time limit: a follow-up may come months later.
SELECT
    e.id,
    e.title,
    ts_rank(e.search, q.query) AS rank
FROM events e,
    (SELECT replace(plainto_tsquery('english', sqlc.arg(search_text)::text)::text, '&', '|')::tsquery AS query) q
WHERE e.search @@ q.query
ORDER BY rank DESC, e.updated_at DESC
LIMIT sqlc.arg(max_results);

-- name: ListEventsSharingEntities :many
-- Candidate events that involve any of the given entities, most shared first.
-- Shared names alone never merge events; this only nominates candidates.
SELECT
    ee.event_id AS id,
    count(*)::bigint AS shared
FROM event_entities ee
WHERE ee.entity_id = ANY(sqlc.arg(entity_ids)::bigint[])
GROUP BY ee.event_id
ORDER BY shared DESC, ee.event_id DESC
LIMIT sqlc.arg(max_results);
