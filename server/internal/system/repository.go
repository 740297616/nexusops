package system

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/740297616/nexusops/server/internal/auth"
	"github.com/740297616/nexusops/server/internal/rbac"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("resource not found")
var ErrInvalidStatus = errors.New("invalid status")

type Repository struct {
	db   *gorm.DB
	rbac *rbac.Repository
}

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db, rbac: rbac.NewRepository(db)} }

func userView(u auth.User) UserView {
	return UserView{u.ID, u.Username, u.DisplayName, u.Status, u.Role, u.CreatedAt, u.UpdatedAt, u.LastLoginAt}
}
func (r *Repository) ListUsers(ctx context.Context, page, size int, keyword, status string) ([]UserView, int64, error) {
	if page < 1 || size < 1 || size > 100 {
		return nil, 0, errors.New("invalid pagination")
	}
	q := r.db.WithContext(ctx).Model(&auth.User{})
	if keyword != "" {
		like := "%" + strings.ToLower(keyword) + "%"
		q = q.Where("(lower(username) LIKE ? OR lower(display_name) LIKE ?)", like, like)
	}
	if status != "" {
		q = q.Where("status=?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []auth.User
	if err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	items := make([]UserView, 0, len(users))
	for _, u := range users {
		items = append(items, userView(u))
	}
	return items, total, nil
}
func (r *Repository) GetUser(ctx context.Context, id int64) (UserView, error) {
	var u auth.User
	if err := r.db.WithContext(ctx).First(&u, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return UserView{}, ErrNotFound
		}
		return UserView{}, err
	}
	return userView(u), nil
}
func (r *Repository) CreateUser(ctx context.Context, u *auth.User, actorID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(u).Error; err != nil {
			return err
		}
		return audit(tx, actorID, "user_created", u.ID, nil)
	})
}
func (r *Repository) UpdateUser(ctx context.Context, id int64, display, status string, actorID int64) error {
	if status != "" {
		var current auth.User
		if err := r.db.WithContext(ctx).First(&current, id).Error; err != nil {
			return err
		}
		if status != current.Status {
			if err := r.SetUserStatus(ctx, id, status, actorID); err != nil {
				return err
			}
			status = current.Status
		}
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var u auth.User
		if err := tx.First(&u, id).Error; err != nil {
			return err
		}
		if display == "" {
			display = u.DisplayName
		}
		if status == "" {
			status = u.Status
		}
		if status != "active" && status != "disabled" {
			return ErrInvalidStatus
		}
		if err := tx.Model(&u).Updates(map[string]any{"display_name": display, "updated_at": time.Now().UTC()}).Error; err != nil {
			return err
		}
		if status != "active" {
			if err := tx.Model(&auth.Session{}).Where("user_id=? AND revoked_at IS NULL", id).Update("revoked_at", time.Now().UTC()).Error; err != nil {
				return err
			}
		}
		return audit(tx, actorID, "user_updated", id, map[string]any{"status": status})
	})
}
func (r *Repository) SetUserStatus(ctx context.Context, id int64, status string, actorID int64) error {
	if status != "active" && status != "disabled" {
		return ErrInvalidStatus
	}
	if err := r.rbac.SetUserStatus(ctx, id, status); err != nil {
		return err
	}
	if status != "active" {
		if err := r.db.WithContext(ctx).Model(&auth.Session{}).Where("user_id=? AND revoked_at IS NULL", id).Update("revoked_at", time.Now().UTC()).Error; err != nil {
			return err
		}
	}
	return r.db.WithContext(ctx).Exec("INSERT INTO auth_audit_events(user_id,event_type,outcome,metadata) VALUES (?, ?, ?, ?::jsonb)", actorID, "user_status_changed", "success", fmt.Sprintf(`{"target_user_id":%d,"status":%q}`, id, status)).Error
}
func (r *Repository) CreatePassword(ctx context.Context, id int64, password string, actorID int64) error {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&auth.User{}).Where("id=?", id).Updates(map[string]any{"password_hash": hash, "updated_at": time.Now().UTC()})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return ErrNotFound
		}
		if err := tx.Model(&auth.Session{}).Where("user_id=? AND revoked_at IS NULL", id).Update("revoked_at", time.Now().UTC()).Error; err != nil {
			return err
		}
		return audit(tx, actorID, "user_password_reset", id, nil)
	})
}
func (r *Repository) ListSessions(ctx context.Context, page, size int, userID *int64) ([]auth.SessionInfo, int64, error) {
	q := r.db.WithContext(ctx).Table("auth_sessions").Where("revoked_at IS NULL AND expires_at > ?", time.Now().UTC())
	if userID != nil {
		q = q.Where("user_id=?", *userID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := []auth.SessionInfo{}
	if err := q.Select("id,created_at,access_expires_at,expires_at,false AS current").Order("created_at DESC").Offset((page - 1) * size).Limit(size).Scan(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
func (r *Repository) RevokeUserSessions(ctx context.Context, userID int64, sessionID *int64, actorID int64) error {
	q := r.db.WithContext(ctx).Model(&auth.Session{}).Where("user_id=? AND revoked_at IS NULL", userID)
	if sessionID != nil {
		q = q.Where("id=?", *sessionID)
	}
	if err := q.Update("revoked_at", time.Now().UTC()).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Exec("INSERT INTO auth_audit_events(user_id,event_type,outcome,metadata) VALUES (?, ?, ?, ?::jsonb)", actorID, "admin_sessions_revoked", "success", fmt.Sprintf(`{"target_user_id":%d}`, userID)).Error
}
func (r *Repository) ListAuthAudit(ctx context.Context, page, size int, userID *int64, event, outcome, from, to string) ([]AuditView, int64, error) {
	q := r.db.WithContext(ctx).Table("auth_audit_events")
	if userID != nil {
		q = q.Where("user_id=?", *userID)
	}
	if event != "" {
		q = q.Where("event_type=?", event)
	}
	if outcome != "" {
		q = q.Where("outcome=?", outcome)
	}
	if from != "" {
		q = q.Where("created_at >= ?", from)
	}
	if to != "" {
		q = q.Where("created_at <= ?", to)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []struct {
		ID                                      int64
		UserID                                  *int64
		EventType, Outcome, SourceIP, UserAgent string
		SessionID                               *int64
		Metadata                                []byte
		CreatedAt                               time.Time
	}
	if err := q.Select("id,user_id,event_type,outcome,host(source_ip) AS source_ip,user_agent,session_id,metadata,created_at").Order("created_at DESC").Offset((page - 1) * size).Limit(size).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]AuditView, 0, len(rows))
	for _, row := range rows {
		m := map[string]any{}
		_ = json.Unmarshal(row.Metadata, &m)
		items = append(items, AuditView{row.ID, row.UserID, row.EventType, row.Outcome, row.SourceIP, row.UserAgent, row.SessionID, m, row.CreatedAt})
	}
	return items, total, nil
}
func (r *Repository) ListRoleAudit(ctx context.Context, page, size int) ([]RoleAuditView, int64, error) {
	q := r.db.WithContext(ctx).Table("rbac_audit_events")
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []struct {
		ID                          int64
		ActorUserID, TargetUserID   *int64
		Action                      string
		PreviousRoleIDs, NewRoleIDs []byte
		CreatedAt                   time.Time
	}
	if err := q.Select("id,actor_user_id,target_user_id,action,previous_role_ids,new_role_ids,created_at").Order("created_at DESC").Offset((page - 1) * size).Limit(size).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]RoleAuditView, 0, len(rows))
	for _, row := range rows {
		var before, after []int64
		_ = json.Unmarshal(row.PreviousRoleIDs, &before)
		_ = json.Unmarshal(row.NewRoleIDs, &after)
		items = append(items, RoleAuditView{row.ID, row.ActorUserID, row.TargetUserID, row.Action, before, after, row.CreatedAt})
	}
	return items, total, nil
}
func (r *Repository) Config(ctx context.Context) ([]ConfigView, error) {
	var rows []struct {
		Key       string
		Value     []byte
		Editable  bool
		UpdatedAt time.Time
	}
	if err := r.db.WithContext(ctx).Table("system_configs").Select("key,value,editable,updated_at").Where("is_sensitive=false").Order("key").Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]ConfigView, 0, len(rows))
	for _, row := range rows {
		var v any
		_ = json.Unmarshal(row.Value, &v)
		items = append(items, ConfigView{row.Key, v, row.Editable, row.UpdatedAt})
	}
	return items, nil
}
func (r *Repository) SetConfig(ctx context.Context, actorID int64, values map[string]json.RawMessage) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for key, raw := range values {
			var editable bool
			var exists bool
			if err := tx.Raw("SELECT EXISTS(SELECT 1 FROM system_configs WHERE key=? AND is_sensitive=false), COALESCE((SELECT editable FROM system_configs WHERE key=? AND is_sensitive=false), false)", key, key).Row().Scan(&exists, &editable); err != nil {
				return err
			}
			if !exists {
				return errors.New("unknown system config")
			}
			if !editable {
				return errors.New("config is not editable")
			}
			if err := tx.Exec("UPDATE system_configs SET value=?::jsonb,updated_by=?,updated_at=now() WHERE key=?", string(raw), actorID, key).Error; err != nil {
				return err
			}
		}
		return audit(tx, actorID, "system_config_updated", 0, map[string]any{"keys": keys(values)})
	})
}
func keys(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
func audit(tx *gorm.DB, actorID int64, event string, target int64, details map[string]any) error {
	body, _ := json.Marshal(map[string]any{"target_id": target, "details": details})
	return tx.Exec("INSERT INTO auth_audit_events(user_id,event_type,outcome,metadata) VALUES (?, ?, ?, ?::jsonb)", actorID, event, "success", string(body)).Error
}
