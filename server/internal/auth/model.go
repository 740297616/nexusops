package auth

import "time"

type User struct {
	ID           int64      `gorm:"primaryKey" json:"id"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"-"`
	DisplayName  string     `json:"display_name"`
	Status       string     `json:"status"`
	Role         string     `json:"role"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at"`
}

type Session struct {
	ID              int64 `gorm:"primaryKey"`
	UserID          int64
	AccessHash      []byte
	AccessExpiresAt time.Time
	ExpiresAt       time.Time
	RevokedAt       *time.Time
	CreatedAt       time.Time
}

func (Session) TableName() string { return "auth_sessions" }

type RefreshToken struct {
	ID         int64 `gorm:"primaryKey"`
	SessionID  int64
	TokenHash  []byte
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	CreatedAt  time.Time
}

func (RefreshToken) TableName() string { return "auth_refresh_tokens" }
