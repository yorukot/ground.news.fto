-- name: ListPublicOutlets :many
SELECT id, slug, name, domain
FROM outlets
WHERE NOT is_aggregator
ORDER BY name;

-- name: UpsertOutlet :one
INSERT INTO outlets (slug, name, domain, is_aggregator, enabled, crawl_config)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (slug) DO UPDATE
SET name = EXCLUDED.name,
    domain = EXCLUDED.domain,
    is_aggregator = EXCLUDED.is_aggregator,
    -- The seed only supplies crawl settings for an outlet that has none, so
    -- settings changed in the database are never overwritten by a re-seed.
    enabled = CASE WHEN outlets.crawl_config = '{}' THEN EXCLUDED.enabled ELSE outlets.enabled END,
    crawl_config = CASE WHEN outlets.crawl_config = '{}' THEN EXCLUDED.crawl_config ELSE outlets.crawl_config END
RETURNING id;

-- name: ListEnabledOutlets :many
SELECT id, slug, name, is_aggregator, crawl_config
FROM outlets
WHERE enabled
ORDER BY id;

-- name: GetOutlet :one
SELECT id, slug, name, is_aggregator, enabled, crawl_config
FROM outlets
WHERE id = $1;

-- name: FindOutletByName :one
-- Resolves the publisher an aggregator names to one of our outlets.
SELECT id
FROM outlets
WHERE name = $1 AND NOT is_aggregator;

-- name: SetOutletCrawl :exec
UPDATE outlets
SET enabled = $2, crawl_config = $3
WHERE slug = $1;
