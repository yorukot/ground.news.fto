-- +goose Up
ALTER TABLE articles ADD COLUMN manual_link boolean NOT NULL DEFAULT false;
-- The old admin always wrote confidence 1; headline matching wrote .5 or .7.
UPDATE articles SET manual_link=true WHERE body='' AND link_confidence=1;
UPDATE articles SET event_id=NULL, step_article_id=NULL, link_confidence=NULL
WHERE body='' AND NOT manual_link AND url NOT LIKE 'https://example.com/ground-sample/%';
UPDATE outlets SET coverage='full';

CREATE TABLE crawl_urls (
    url text PRIMARY KEY,
    outlet_id bigint NOT NULL REFERENCES outlets(id),
    source text NOT NULL DEFAULT '',
    headline text NOT NULL DEFAULT '',
    image_url text NOT NULL DEFAULT '',
    category text NOT NULL DEFAULT '',
    scope_reason text NOT NULL DEFAULT 'unknown section',
    eligible boolean NOT NULL DEFAULT true,
    published_at timestamptz,
    first_seen_at timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    result text NOT NULL DEFAULT 'pending',
    http_status integer NOT NULL DEFAULT 0,
    error text NOT NULL DEFAULT '',
    extractor text NOT NULL DEFAULT '',
    body_length integer NOT NULL DEFAULT 0,
    fetched_at timestamptz,
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    article_id bigint REFERENCES articles(id) ON DELETE SET NULL
);
CREATE INDEX crawl_urls_pending_idx ON crawl_urls(next_attempt_at,first_seen_at) WHERE result IN ('pending','retry');
CREATE TABLE crawl_sources (
    outlet_id bigint NOT NULL REFERENCES outlets(id),
    source text NOT NULL,
    mode text NOT NULL DEFAULT 'live',
    since_at timestamptz NOT NULL,
    progress jsonb NOT NULL DEFAULT '{}',
    error text NOT NULL DEFAULT '',
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY(outlet_id,source,mode)
);
CREATE TABLE ai_admissions (
    article_id bigint PRIMARY KEY REFERENCES articles(id) ON DELETE CASCADE,
    outlet_id bigint NOT NULL REFERENCES outlets(id),
    admitted_on date NOT NULL,
    admitted_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ai_admissions_day_idx ON ai_admissions(admitted_on);
CREATE INDEX ai_admissions_outlet_idx ON ai_admissions(outlet_id,admitted_on,admitted_at);
CREATE INDEX crawl_urls_article_idx ON crawl_urls(article_id);
CREATE TABLE analysis_pending (
    article_id bigint PRIMARY KEY REFERENCES articles(id) ON DELETE CASCADE,
    ready_at timestamptz NOT NULL DEFAULT now()
);

-- Preserve actual old coverage; sample URLs are never sent to the crawler.
INSERT INTO crawl_urls(url,outlet_id,headline,published_at,article_id,result,body_length,fetched_at)
SELECT url,outlet_id,headline,published_at,id,
       CASE WHEN body='' THEN 'pending' ELSE 'success' END,char_length(body),
       CASE WHEN body='' THEN NULL ELSE first_seen_at END
FROM articles WHERE url NOT LIKE 'https://example.com/ground-sample/%'
AND (body<>'' OR published_at>=now()-interval '7 days');

INSERT INTO analysis_pending(article_id)
SELECT a.id FROM articles a
WHERE a.body<>'' AND a.reprint_of_id IS NULL
AND a.url NOT LIKE 'https://example.com/ground-sample/%'
AND NOT EXISTS(SELECT 1 FROM summaries s WHERE s.article_id=a.id);

-- +goose Down
DROP TABLE analysis_pending, ai_admissions, crawl_sources, crawl_urls;
ALTER TABLE articles DROP COLUMN manual_link;
