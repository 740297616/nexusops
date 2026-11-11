package system

import "time"

type Page struct {
	Items    any   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}
type UserView struct {
	ID          int64      `json:"id"`
	Username    string     `json:"username"`
	DisplayName string     `json:"display_name"`
	Status      string     `json:"status"`
	Role        string     `json:"role"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastLoginAt *time.Time `json:"last_login_at"`
}
type AuditView struct {
	ID        int64          `json:"id"`
	UserID    *int64         `json:"user_id"`
	EventType string         `json:"event_type"`
	Outcome   string         `json:"outcome"`
	SourceIP  string         `json:"source_ip"`
	UserAgent string         `json:"user_agent"`
	SessionID *int64         `json:"session_id"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
}
type RoleAuditView struct {
	ID              int64     `json:"id"`
	ActorUserID     *int64    `json:"actor_user_id"`
	TargetUserID    *int64    `json:"target_user_id"`
	Action          string    `json:"action"`
	PreviousRoleIDs []int64   `json:"previous_role_ids"`
	NewRoleIDs      []int64   `json:"new_role_ids"`
	CreatedAt       time.Time `json:"created_at"`
}
type ConfigView struct {
	Key       string    `json:"key"`
	Value     any       `json:"value"`
	Editable  bool      `json:"editable"`
	UpdatedAt time.Time `json:"updated_at"`
}
