-- +goose Up
CREATE TABLE auth_audit_events (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    event_type VARCHAR(64) NOT NULL,
    outcome VARCHAR(32) NOT NULL CHECK (outcome IN ('success', 'failure', 'denied')),
    source_ip INET,
    user_agent TEXT NOT NULL DEFAULT '',
    session_id BIGINT REFERENCES auth_sessions(id) ON DELETE SET NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX auth_audit_events_user_created_idx ON auth_audit_events(user_id, created_at DESC);
CREATE INDEX auth_audit_events_type_created_idx ON auth_audit_events(event_type, created_at DESC);
CREATE INDEX auth_audit_events_created_idx ON auth_audit_events(created_at);

CREATE TABLE auth_rate_limits (
    rate_key VARCHAR(256) PRIMARY KEY,
    window_started_at TIMESTAMPTZ NOT NULL,
    attempts INTEGER NOT NULL CHECK (attempts > 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX auth_rate_limits_updated_idx ON auth_rate_limits(updated_at);

-- +goose Down
DROP TABLE auth_rate_limits;
DROP TABLE auth_audit_events;
