-- name: ListEventArticles :many
-- Every article of an event with its outlet and summary. The body is
-- deliberately not selected: article text is never served.
SELECT
    a.id,
    a.url,
    a.headline,
    a.image_url,
    a.published_at,
    a.reprint_of_id,
    a.development,
    a.happened_on,
    a.date_is_approximate,
    a.step_article_id,
    o.id AS outlet_id,
    o.slug AS outlet_slug,
    o.name AS outlet_name,
    (o.coverage = 'headline')::boolean AS headline_only,
    COALESCE(s.text, '')::text AS summary
FROM articles a
JOIN outlets o ON o.id = a.outlet_id
LEFT JOIN summaries s ON s.article_id = a.id
WHERE a.event_id = sqlc.arg(event_id)::bigint
ORDER BY a.published_at DESC, a.id DESC;

-- name: CreateArticle :one
INSERT INTO articles (
    outlet_id, event_id, url, headline, body, published_at, first_seen_at,
    content_hash, reprint_of_id, development, happened_on, date_is_approximate
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING id;

-- name: SetArticleStep :exec
UPDATE articles SET step_article_id = $2 WHERE id = $1;

-- name: UpsertSummary :exec
INSERT INTO summaries (article_id, text, recaps, model, prompt_version)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (article_id) DO UPDATE
SET text = EXCLUDED.text,
    recaps = EXCLUDED.recaps,
    model = EXCLUDED.model,
    prompt_version = EXCLUDED.prompt_version,
    created_at = now();
