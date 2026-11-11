-- +goose Up
CREATE TABLE roles (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(128) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE permissions (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code VARCHAR(128) NOT NULL UNIQUE,
    name VARCHAR(128) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    resource_type VARCHAR(64) NOT NULL,
    action_type VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE user_roles (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, role_id)
);
CREATE INDEX user_roles_role_id_idx ON user_roles(role_id);
CREATE TABLE role_permissions (
    role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id BIGINT NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (role_id, permission_id)
);
CREATE INDEX role_permissions_permission_id_idx ON role_permissions(permission_id);
CREATE TABLE rbac_audit_events (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    target_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    previous_role_ids JSONB NOT NULL,
    new_role_ids JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO roles (code, name, description) VALUES
    ('super_admin', 'Super administrator', 'Full system permissions'),
    ('admin', 'Administrator', 'Routine system administration'),
    ('operator', 'Operator', 'Limited operational access')
ON CONFLICT (code) DO NOTHING;
INSERT INTO permissions (code, name, description, resource_type, action_type) VALUES
    ('system:user:read', 'Read users', 'Read user role assignments', 'system:user', 'read'),
    ('system:user:create', 'Create users', 'Create users when supported', 'system:user', 'create'),
    ('system:user:update', 'Update users', 'Update users when supported', 'system:user', 'update'),
    ('system:user:delete', 'Delete users', 'Delete users when supported', 'system:user', 'delete'),
    ('system:role:read', 'Read roles', 'List roles', 'system:role', 'read'),
    ('system:role:manage', 'Manage roles', 'Assign roles to users', 'system:role', 'manage'),
    ('system:permission:read', 'Read permissions', 'List permissions', 'system:permission', 'read')
ON CONFLICT (code) DO NOTHING;
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p WHERE r.code = 'super_admin'
ON CONFLICT DO NOTHING;
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code IN ('system:user:read', 'system:user:create', 'system:user:update', 'system:role:read', 'system:permission:read') WHERE r.code = 'admin'
ON CONFLICT DO NOTHING;
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.code = 'system:role:read' WHERE r.code = 'operator'
ON CONFLICT DO NOTHING;
INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id FROM users u JOIN roles r ON r.code = 'super_admin' WHERE u.role = 'admin'
ON CONFLICT DO NOTHING;

-- +goose Down
DROP TABLE rbac_audit_events;
DROP TABLE role_permissions;
DROP TABLE user_roles;
DROP TABLE permissions;
DROP TABLE roles;
