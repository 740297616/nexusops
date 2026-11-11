package rbac

import "time"

const (
	PermissionUserRead         = "system:user:read"
	PermissionUserCreate       = "system:user:create"
	PermissionUserUpdate       = "system:user:update"
	PermissionUserDelete       = "system:user:delete"
	PermissionRoleRead         = "system:role:read"
	PermissionRoleManage       = "system:role:manage"
	PermissionPermissionRead   = "system:permission:read"
	PermissionRoleCreate       = "system:role:create"
	PermissionRoleUpdate       = "system:role:update"
	PermissionRoleDelete       = "system:role:delete"
	PermissionPermissionManage = "system:permission:manage"
	PermissionAuditRead        = "system:audit:read"
	PermissionSessionRead      = "system:session:read"
	PermissionSessionManage    = "system:session:manage"
	PermissionConfigRead       = "system:config:read"
	PermissionConfigUpdate     = "system:config:update"
	PermissionServerRead       = "infrastructure:server:read"
	PermissionServerCreate     = "infrastructure:server:create"
	PermissionServerUpdate     = "infrastructure:server:update"
	PermissionServerDelete     = "infrastructure:server:delete"
	PermissionServerAgent      = "infrastructure:server:manage-agent"
)

type Role struct {
	ID          int64     `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	IsSystem    bool      `json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
type Permission struct {
	ID           int64     `json:"id"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	ResourceType string    `json:"resource_type"`
	ActionType   string    `json:"action_type"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
