-- +goose Up
-- Reader-facing overview synthesized from all full-coverage articles in an event.
ALTER TABLE events
    ADD COLUMN summary text NOT NULL DEFAULT '',
    ADD COLUMN summary_model text NOT NULL DEFAULT '',
    ADD COLUMN summary_prompt_version text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE events
    DROP COLUMN summary_prompt_version,
    DROP COLUMN summary_model,
    DROP COLUMN summary;
