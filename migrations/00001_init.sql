-- +goose Up
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE outlets (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    slug          text NOT NULL UNIQUE,
    name          text NOT NULL,
    domain        text NOT NULL,
    -- Aggregators (Yahoo News Taiwan) are crawled only to find articles;
    -- each article is credited to its original outlet.
    is_aggregator boolean NOT NULL DEFAULT false,
    enabled       boolean NOT NULL DEFAULT false,
    crawl_config  jsonb NOT NULL DEFAULT '{}',
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE events (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title         text NOT NULL,
    first_seen_at timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    -- Every timeline line of the event, newline-joined; rewritten by the
    -- application whenever a step is added. With the title it feeds the
    -- full-text search that finds candidate events for a new article. Titles
    -- and timeline lines are the site's own English text, so the built-in
    -- English configuration fits.
    timeline_text text NOT NULL DEFAULT '',
    search        tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('english', title), 'A') ||
        setweight(to_tsvector('english', timeline_text), 'B')
    ) STORED
);

CREATE INDEX events_updated_at_idx ON events (updated_at DESC, id DESC);
CREATE INDEX events_search_idx ON events USING gin (search);

-- One row per article. A timeline step is an article: the first article to
-- report a development has step_article_id = its own id, and later articles
-- reporting the same development point at it.
CREATE TABLE articles (
    id                  bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    outlet_id           bigint NOT NULL REFERENCES outlets (id),
    event_id            bigint REFERENCES events (id) ON DELETE SET NULL,
    url                 text NOT NULL UNIQUE,
    headline            text NOT NULL,
    -- Stored for processing only; never served by the API.
    body                text NOT NULL,
    published_at        timestamptz NOT NULL,
    first_seen_at       timestamptz NOT NULL DEFAULT now(),
    content_hash        text NOT NULL,
    minhash             integer[] NOT NULL DEFAULT '{}',
    reprint_of_id       bigint REFERENCES articles (id) ON DELETE SET NULL,
    development         text NOT NULL DEFAULT '',
    happened_on         date,
    date_is_approximate boolean NOT NULL DEFAULT false,
    step_article_id     bigint REFERENCES articles (id) ON DELETE SET NULL,
    link_confidence     real
);

CREATE INDEX articles_event_idx ON articles (event_id, published_at DESC);
CREATE INDEX articles_outlet_idx ON articles (outlet_id, published_at DESC);
CREATE INDEX articles_reprint_idx ON articles (reprint_of_id) WHERE reprint_of_id IS NOT NULL;
CREATE INDEX articles_first_seen_idx ON articles (first_seen_at DESC);
CREATE INDEX articles_content_hash_idx ON articles (content_hash);

CREATE TABLE summaries (
    article_id     bigint PRIMARY KEY REFERENCES articles (id) ON DELETE CASCADE,
    text           text NOT NULL,
    -- Earlier steps the article recaps; the strongest evidence for linking.
    recaps         text[] NOT NULL DEFAULT '{}',
    model          text NOT NULL,
    prompt_version text NOT NULL,
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE entities (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    canonical_name text NOT NULL UNIQUE,
    -- Places matter for events that name no one: typhoons, earthquakes, local crime.
    kind           text NOT NULL CHECK (kind IN ('person', 'organization', 'place')),
    aliases        text[] NOT NULL DEFAULT '{}',
    -- Canonical name plus aliases, space-joined by the application, for trigram search.
    search_text    text NOT NULL DEFAULT ''
);

CREATE INDEX entities_search_idx ON entities USING gin (search_text gin_trgm_ops);

CREATE TABLE event_entities (
    event_id  bigint NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    entity_id bigint NOT NULL REFERENCES entities (id) ON DELETE CASCADE,
    PRIMARY KEY (event_id, entity_id)
);

CREATE INDEX event_entities_entity_idx ON event_entities (entity_id);

CREATE TABLE article_entities (
    article_id bigint NOT NULL REFERENCES articles (id) ON DELETE CASCADE,
    entity_id  bigint NOT NULL REFERENCES entities (id) ON DELETE CASCADE,
    PRIMARY KEY (article_id, entity_id)
);

-- +goose Down
DROP TABLE article_entities;
DROP TABLE event_entities;
DROP TABLE entities;
DROP TABLE summaries;
DROP TABLE articles;
DROP TABLE events;
DROP TABLE outlets;
