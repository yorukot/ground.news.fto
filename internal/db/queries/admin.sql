-- Queries used by the admin tool to review and correct event linking.

-- name: ListLinkDecisions :many
-- Linked articles, least confident first, for review.
SELECT
    a.id, a.headline, a.published_at, a.link_confidence,
    o.name AS outlet_name,
    e.id AS event_id, e.title AS event_title
FROM articles a
JOIN outlets o ON o.id = a.outlet_id
JOIN events e ON e.id = a.event_id
WHERE a.link_confidence IS NOT NULL
  AND a.link_confidence <= sqlc.arg(max_confidence)
ORDER BY a.link_confidence, a.id DESC
LIMIT sqlc.arg(max_results);

-- name: SearchEventsByTitle :many
-- Finds the target event for a merge or a move. Matches the title loosely, or
-- an exact id typed as a number.
SELECT e.id, e.title, e.updated_at,
    (SELECT count(*) FROM articles a WHERE a.event_id = e.id)::bigint AS article_count
FROM events e
WHERE e.title ILIKE '%' || sqlc.arg(query)::text || '%'
   OR e.id::text = sqlc.arg(query)::text
ORDER BY e.updated_at DESC
LIMIT sqlc.arg(max_results);

-- name: GetArticleForMove :one
SELECT id, event_id, step_article_id, development, reprint_of_id
FROM articles
WHERE id = $1
FOR UPDATE;

-- name: ListStepFollowers :many
-- Articles in an event that attach to the given step, other than the step
-- article itself, earliest first.
SELECT id, development
FROM articles
WHERE step_article_id = sqlc.arg(step_article_id)
  AND id <> sqlc.arg(step_article_id)
  AND event_id = sqlc.arg(event_id)
ORDER BY published_at, id;

-- name: RepointStep :exec
UPDATE articles
SET step_article_id = sqlc.arg(new_step_id)
WHERE step_article_id = sqlc.arg(old_step_id) AND event_id = sqlc.arg(event_id);

-- name: MoveArticle :exec
-- An admin's decision is certain, so the confidence becomes 1.
UPDATE articles
SET event_id = $2, step_article_id = $3, link_confidence = 1, manual_link = true
WHERE id = $1;

-- name: MoveEventArticles :exec
UPDATE articles SET event_id = sqlc.arg(to_event_id), manual_link=true WHERE event_id = sqlc.arg(from_event_id);

-- name: CopyEventEntities :exec
INSERT INTO event_entities (event_id, entity_id)
SELECT sqlc.arg(to_event_id), ee.entity_id FROM event_entities ee WHERE ee.event_id = sqlc.arg(from_event_id)
ON CONFLICT DO NOTHING;

-- name: CopyArticleEntitiesToEvent :exec
INSERT INTO event_entities (event_id, entity_id)
SELECT sqlc.arg(event_id), ae.entity_id FROM article_entities ae WHERE ae.article_id = sqlc.arg(article_id)
ON CONFLICT DO NOTHING;

-- name: LockEvent :one
SELECT id, first_seen_at FROM events WHERE id = $1 FOR UPDATE;

-- name: DeleteEvent :exec
DELETE FROM events WHERE id = $1;

-- name: DeleteEventIfEmpty :exec
DELETE FROM events e
WHERE e.id = $1 AND NOT EXISTS (SELECT 1 FROM articles a WHERE a.event_id = e.id);

-- name: SetEventFirstSeen :exec
UPDATE events SET first_seen_at = LEAST(first_seen_at, sqlc.arg(first_seen_at)) WHERE id = sqlc.arg(id);
