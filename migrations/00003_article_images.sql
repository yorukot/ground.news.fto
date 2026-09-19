-- +goose Up
-- Keep only the publisher's image URL. The image itself remains at the source.
ALTER TABLE articles ADD COLUMN image_url text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE articles DROP COLUMN image_url;
