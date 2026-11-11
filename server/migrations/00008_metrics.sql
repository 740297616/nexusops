-- +goose Up
CREATE TABLE metric_samples (
    id BIGSERIAL PRIMARY KEY,
    server_id VARCHAR(36) NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    agent_credential_id BIGINT REFERENCES agent_credentials(id) ON DELETE SET NULL,
    request_id VARCHAR(128) NOT NULL,
    schema_version INTEGER NOT NULL,
    sampled_at TIMESTAMPTZ NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (server_id, request_id)
);
CREATE INDEX metric_samples_server_sampled_idx ON metric_samples(server_id, sampled_at DESC);
CREATE INDEX metric_samples_received_idx ON metric_samples(received_at DESC);

CREATE TABLE metric_values (
    sample_id BIGINT NOT NULL REFERENCES metric_samples(id) ON DELETE CASCADE,
    metric_type VARCHAR(32) NOT NULL,
    metric_name VARCHAR(64) NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    unit VARCHAR(32) NOT NULL,
    PRIMARY KEY (sample_id, metric_type, metric_name)
);
CREATE INDEX metric_values_type_name_idx ON metric_values(metric_type, metric_name, sample_id);

CREATE TABLE metric_filesystems (
    sample_id BIGINT NOT NULL REFERENCES metric_samples(id) ON DELETE CASCADE,
    mountpoint VARCHAR(255) NOT NULL,
    filesystem VARCHAR(64) NOT NULL DEFAULT '',
    metric_name VARCHAR(64) NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    unit VARCHAR(32) NOT NULL,
    PRIMARY KEY (sample_id, mountpoint, metric_name)
);
CREATE INDEX metric_filesystems_mountpoint_idx ON metric_filesystems(mountpoint, sample_id);

CREATE TABLE metric_networks (
    sample_id BIGINT NOT NULL REFERENCES metric_samples(id) ON DELETE CASCADE,
    interface_name VARCHAR(128) NOT NULL,
    metric_name VARCHAR(64) NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    unit VARCHAR(32) NOT NULL,
    PRIMARY KEY (sample_id, interface_name, metric_name)
);
CREATE INDEX metric_networks_interface_idx ON metric_networks(interface_name, sample_id);

-- +goose Down
DROP TABLE metric_networks;
DROP TABLE metric_filesystems;
DROP TABLE metric_values;
DROP TABLE metric_samples;
