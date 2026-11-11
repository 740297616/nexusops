-- +goose Up
CREATE TABLE agent_rate_limits (
    rate_key VARCHAR(255) PRIMARY KEY,
    window_started_at TIMESTAMPTZ NOT NULL,
    attempts INTEGER NOT NULL CHECK (attempts >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX agent_rate_limits_updated_idx ON agent_rate_limits(updated_at);

INSERT INTO permissions (code, name, description, resource_type, action_type) VALUES
    ('infrastructure:server:group-manage', 'Manage server groups', 'Create and assign server groups', 'infrastructure:server', 'group-manage'),
    ('infrastructure:server:tag-manage', 'Manage server tags', 'Create and assign server tags', 'infrastructure:server', 'tag-manage')
ON CONFLICT (code) DO NOTHING;
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p WHERE r.code = 'super_admin' AND p.code IN ('infrastructure:server:group-manage', 'infrastructure:server:tag-manage') ON CONFLICT DO NOTHING;
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code IN ('infrastructure:server:group-manage', 'infrastructure:server:tag-manage') WHERE r.code = 'admin' ON CONFLICT DO NOTHING;

-- +goose Down
DROP TABLE agent_rate_limits;
DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code IN ('infrastructure:server:group-manage', 'infrastructure:server:tag-manage'));
DELETE FROM permissions WHERE code IN ('infrastructure:server:group-manage', 'infrastructure:server:tag-manage');
