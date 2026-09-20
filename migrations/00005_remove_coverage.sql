-- +goose Up
ALTER TABLE outlets DROP COLUMN coverage;
CREATE INDEX IF NOT EXISTS ai_admissions_outlet_idx ON ai_admissions(outlet_id,admitted_on,admitted_at);
CREATE INDEX IF NOT EXISTS crawl_urls_article_idx ON crawl_urls(article_id);

-- +goose Down
ALTER TABLE outlets ADD COLUMN coverage text NOT NULL DEFAULT 'full' CHECK(coverage IN ('full','headline'));
