-- +goose Up
CREATE TABLE servers (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    hostname VARCHAR(255) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    os VARCHAR(128) NOT NULL DEFAULT '',
    os_version VARCHAR(128) NOT NULL DEFAULT '',
    architecture VARCHAR(64) NOT NULL DEFAULT '',
    kernel_version VARCHAR(128) NOT NULL DEFAULT '',
    cpu_model VARCHAR(255) NOT NULL DEFAULT '',
    cpu_cores INTEGER NOT NULL DEFAULT 0 CHECK (cpu_cores >= 0),
    memory_total BIGINT NOT NULL DEFAULT 0 CHECK (memory_total >= 0),
    disk_total BIGINT NOT NULL DEFAULT 0 CHECK (disk_total >= 0),
    primary_ip VARCHAR(128) NOT NULL DEFAULT '',
    agent_version VARCHAR(128) NOT NULL DEFAULT '',
    agent_status VARCHAR(32) NOT NULL DEFAULT 'unregistered',
    last_heartbeat_at TIMESTAMPTZ,
    last_report_at TIMESTAMPTZ,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT servers_status_check CHECK (status IN ('pending', 'online', 'offline', 'disabled')),
    CONSTRAINT servers_agent_status_check CHECK (agent_status IN ('unregistered', 'online', 'offline', 'revoked'))
);
CREATE INDEX servers_status_idx ON servers(status);
CREATE INDEX servers_agent_status_idx ON servers(agent_status);
CREATE INDEX servers_heartbeat_idx ON servers(last_heartbeat_at DESC);
CREATE INDEX servers_hostname_idx ON servers(hostname);

CREATE TABLE server_groups (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(128) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE server_group_members (
    server_id VARCHAR(36) NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES server_groups(id) ON DELETE CASCADE,
    PRIMARY KEY (server_id, group_id)
);
CREATE INDEX server_group_members_group_idx ON server_group_members(group_id);

CREATE TABLE server_tags (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE server_tag_relations (
    server_id VARCHAR(36) NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    tag_id BIGINT NOT NULL REFERENCES server_tags(id) ON DELETE CASCADE,
    PRIMARY KEY (server_id, tag_id)
);
CREATE INDEX server_tag_relations_tag_idx ON server_tag_relations(tag_id);

CREATE TABLE agent_registration_tokens (
    id BIGSERIAL PRIMARY KEY,
    server_id VARCHAR(36) NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    token_hash BYTEA NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX agent_registration_tokens_server_idx ON agent_registration_tokens(server_id, created_at DESC);

CREATE TABLE agent_credentials (
    id BIGSERIAL PRIMARY KEY,
    server_id VARCHAR(36) NOT NULL UNIQUE REFERENCES servers(id) ON DELETE CASCADE,
    credential_hash BYTEA NOT NULL UNIQUE,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    issued_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    rotated_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    CONSTRAINT agent_credentials_status_check CHECK (status IN ('active', 'revoked'))
);
CREATE INDEX agent_credentials_status_idx ON agent_credentials(status);

INSERT INTO permissions (code, name, description, resource_type, action_type) VALUES
    ('infrastructure:server:read', 'Read servers', 'View server assets and Agent status', 'infrastructure:server', 'read'),
    ('infrastructure:server:create', 'Create servers', 'Create server assets', 'infrastructure:server', 'create'),
    ('infrastructure:server:update', 'Update servers', 'Update server asset metadata', 'infrastructure:server', 'update'),
    ('infrastructure:server:delete', 'Delete servers', 'Delete server assets', 'infrastructure:server', 'delete'),
    ('infrastructure:server:manage-agent', 'Manage Agents', 'Register, rotate and revoke Agents', 'infrastructure:server', 'manage-agent')
ON CONFLICT (code) DO NOTHING;
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p WHERE r.code = 'super_admin'
  AND p.code LIKE 'infrastructure:server:%' ON CONFLICT DO NOTHING;
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code IN (
    'infrastructure:server:read', 'infrastructure:server:create', 'infrastructure:server:update', 'infrastructure:server:manage-agent'
) WHERE r.code = 'admin' ON CONFLICT DO NOTHING;
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code = 'infrastructure:server:read'
WHERE r.code = 'operator' ON CONFLICT DO NOTHING;

-- +goose Down
DROP TABLE agent_credentials;
DROP TABLE agent_registration_tokens;
DROP TABLE server_tag_relations;
DROP TABLE server_tags;
DROP TABLE server_group_members;
DROP TABLE server_groups;
DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code LIKE 'infrastructure:server:%');
DELETE FROM permissions WHERE code LIKE 'infrastructure:server:%';
DROP TABLE servers;
