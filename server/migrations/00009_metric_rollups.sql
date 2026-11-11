-- +goose Up
CREATE TABLE metric_rollups (
    server_id VARCHAR(36) NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    bucket_start TIMESTAMPTZ NOT NULL,
    step_seconds INTEGER NOT NULL CHECK (step_seconds IN (300, 3600)),
    metric_type VARCHAR(32) NOT NULL,
    metric_name VARCHAR(64) NOT NULL,
    labels JSONB NOT NULL DEFAULT '{}'::jsonb,
    avg_value DOUBLE PRECISION NOT NULL,
    min_value DOUBLE PRECISION NOT NULL,
    max_value DOUBLE PRECISION NOT NULL,
    last_value DOUBLE PRECISION NOT NULL,
    last_sampled_at TIMESTAMPTZ NOT NULL,
    last_received_at TIMESTAMPTZ NOT NULL,
    unit VARCHAR(32) NOT NULL,
    sample_count BIGINT NOT NULL CHECK (sample_count > 0),
    PRIMARY KEY (server_id, bucket_start, step_seconds, metric_type, metric_name, labels)
);
CREATE INDEX metric_rollups_query_idx ON metric_rollups(server_id, step_seconds, metric_type, metric_name, bucket_start);

-- +goose Down
DROP TABLE metric_rollups;
