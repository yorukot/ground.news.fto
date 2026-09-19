-- +goose Up
-- How much of an outlet's articles the site may use.
--   full:     the article page is fetched and given to the model (summary, timeline, linking).
--   headline: only feed metadata is stored (headline, link, time). The page is never fetched
--             and nothing is given to a model. For outlets that refuse AI use of their content.
ALTER TABLE outlets
    ADD COLUMN coverage text NOT NULL DEFAULT 'full' CHECK (coverage IN ('full', 'headline'));

-- +goose Down
ALTER TABLE outlets DROP COLUMN coverage;
