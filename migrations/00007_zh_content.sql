-- +goose Up
-- Traditional Chinese versions of the site's own text. English stays the
-- working language (linking, full-text search); these are for display, and an
-- empty value means "not translated yet", in which case the API serves English.
ALTER TABLE events    ADD COLUMN title_zh       text NOT NULL DEFAULT '';
ALTER TABLE events    ADD COLUMN summary_zh     text NOT NULL DEFAULT '';
ALTER TABLE articles  ADD COLUMN development_zh text NOT NULL DEFAULT '';
ALTER TABLE summaries ADD COLUMN text_zh        text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE summaries DROP COLUMN text_zh;
ALTER TABLE articles  DROP COLUMN development_zh;
ALTER TABLE events    DROP COLUMN summary_zh;
ALTER TABLE events    DROP COLUMN title_zh;
