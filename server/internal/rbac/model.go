package rbac

import "time"

const (
	PermissionUserRead       = "system:user:read"
	PermissionUserCreate     = "system:user:create"
	PermissionUserUpdate     = "system:user:update"
	PermissionUserDelete     = "system:user:delete"
	PermissionRoleRead       = "system:role:read"
	PermissionRoleManage     = "system:role:manage"
	PermissionPermissionRead = "system:permission:read"
)

type Role struct {
	ID          int64     `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
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
