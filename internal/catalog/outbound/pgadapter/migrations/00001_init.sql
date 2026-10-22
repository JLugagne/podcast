-- +goose Up
CREATE TABLE catalog_series (
    id                 UUID PRIMARY KEY,
    slug               TEXT        NOT NULL,
    title              TEXT        NOT NULL,
    author             TEXT        NOT NULL,
    description        TEXT        NOT NULL,
    accent_color       TEXT        NOT NULL,
    cover_key          TEXT,
    cover_content_type TEXT,
    cover_size_bytes   BIGINT,
    created_at         TIMESTAMPTZ NOT NULL,
    updated_at         TIMESTAMPTZ NOT NULL,
    version            INTEGER     NOT NULL DEFAULT 0,
    CONSTRAINT catalog_series_slug_key UNIQUE (slug)
);

CREATE INDEX catalog_series_created_at_idx ON catalog_series (created_at DESC, id DESC);

CREATE TABLE catalog_episodes (
    id                     UUID PRIMARY KEY,
    series_id              UUID        NOT NULL,
    number                 INTEGER     NOT NULL,
    title                  TEXT        NOT NULL,
    description            TEXT        NOT NULL,
    status                 TEXT        NOT NULL,
    audio_key              TEXT,
    audio_content_type     TEXT,
    audio_size_bytes       BIGINT,
    audio_duration_seconds DOUBLE PRECISION,
    audio_original_name    TEXT,
    chapters               JSONB       NOT NULL DEFAULT '[]'::jsonb,
    published_at           TIMESTAMPTZ,
    created_at             TIMESTAMPTZ NOT NULL,
    updated_at             TIMESTAMPTZ NOT NULL,
    version                INTEGER     NOT NULL DEFAULT 0,
    CONSTRAINT catalog_episodes_series_number_key UNIQUE (series_id, number),
    CONSTRAINT catalog_episodes_series_id_fkey FOREIGN KEY (series_id)
        REFERENCES catalog_series (id) ON DELETE RESTRICT,
    CONSTRAINT catalog_episodes_status_check CHECK (status IN ('draft', 'published'))
);

CREATE INDEX catalog_episodes_series_id_number_idx ON catalog_episodes (series_id, number);
CREATE INDEX catalog_episodes_series_status_idx ON catalog_episodes (series_id, status, number);

-- +goose Down
DROP TABLE catalog_episodes;
DROP TABLE catalog_series;
