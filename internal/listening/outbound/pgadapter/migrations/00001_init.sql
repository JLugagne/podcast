-- +goose Up
CREATE TABLE listening_progress (
    listener_id      TEXT             NOT NULL,
    episode_id       UUID             NOT NULL,
    position_seconds DOUBLE PRECISION NOT NULL,
    completed        BOOLEAN          NOT NULL,
    created_at       TIMESTAMPTZ      NOT NULL,
    updated_at       TIMESTAMPTZ      NOT NULL,
    PRIMARY KEY (listener_id, episode_id)
);

CREATE INDEX listening_progress_listener_updated_idx
    ON listening_progress (listener_id, updated_at DESC);

-- +goose Down
DROP TABLE listening_progress;
