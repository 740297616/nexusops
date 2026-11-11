-- +goose Up
ALTER TABLE roles ADD COLUMN IF NOT EXISTS is_system BOOLEAN NOT NULL DEFAULT false;
UPDATE roles SET is_system = true WHERE code IN ('super_admin', 'admin', 'operator');

CREATE TABLE system_configs (
    key VARCHAR(128) PRIMARY KEY,
    value JSONB NOT NULL,
    is_sensitive BOOLEAN NOT NULL DEFAULT false,
    editable BOOLEAN NOT NULL DEFAULT true,
    updated_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO permissions (code, name, description, resource_type, action_type) VALUES
    ('system:role:create', 'Create roles', 'Create custom roles', 'system:role', 'create'),
    ('system:role:update', 'Update roles', 'Update custom roles', 'system:role', 'update'),
    ('system:role:delete', 'Delete roles', 'Delete custom roles', 'system:role', 'delete'),
    ('system:permission:manage', 'Manage permissions', 'Assign permissions to roles', 'system:permission', 'manage'),
    ('system:audit:read', 'Read audit events', 'Query security audit events', 'system:audit', 'read'),
    ('system:session:read', 'Read sessions', 'Query user sessions', 'system:session', 'read'),
    ('system:session:manage', 'Manage sessions', 'Revoke user sessions', 'system:session', 'manage'),
    ('system:config:read', 'Read system config', 'Read non-sensitive configuration', 'system:config', 'read'),
    ('system:config:update', 'Update system config', 'Update editable configuration', 'system:config', 'update')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p WHERE r.code = 'super_admin'
ON CONFLICT DO NOTHING;
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code IN ('system:user:read', 'system:user:create', 'system:user:update', 'system:role:read', 'system:role:create', 'system:role:update', 'system:permission:read', 'system:audit:read', 'system:session:read', 'system:session:manage', 'system:config:read') WHERE r.code = 'admin'
ON CONFLICT DO NOTHING;

CREATE INDEX auth_audit_events_outcome_created_idx ON auth_audit_events(outcome, created_at DESC);
CREATE INDEX auth_audit_events_session_created_idx ON auth_audit_events(session_id, created_at DESC);

-- +goose Down
DROP INDEX auth_audit_events_session_created_idx;
DROP INDEX auth_audit_events_outcome_created_idx;
DROP TABLE system_configs;
DELETE FROM role_permissions WHERE permission_id IN (SELECT id FROM permissions WHERE code IN ('system:role:create','system:role:update','system:role:delete','system:permission:manage','system:audit:read','system:session:read','system:session:manage','system:config:read','system:config:update'));
DELETE FROM permissions WHERE code IN ('system:role:create','system:role:update','system:role:delete','system:permission:manage','system:audit:read','system:session:read','system:session:manage','system:config:read','system:config:update');
ALTER TABLE roles DROP COLUMN is_system;
