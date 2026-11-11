package rbac

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"gorm.io/gorm"
)

var ErrUserNotFound = errors.New("user not found")
var ErrInvalidRole = errors.New("unknown or disabled role")
var ErrLastSuperAdmin = errors.New("cannot remove the last active super administrator")

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) HasPermission(ctx context.Context, userID int64, permission string) (bool, error) {
	var allowed bool
	err := r.db.WithContext(ctx).Raw(`SELECT EXISTS (
		SELECT 1 FROM users u JOIN user_roles ur ON ur.user_id = u.id
		JOIN roles r ON r.id = ur.role_id JOIN role_permissions rp ON rp.role_id = r.id
		JOIN permissions p ON p.id = rp.permission_id
		WHERE u.id = ? AND u.status = 'active' AND r.status = 'active' AND p.code = ?
	)`, userID, permission).Scan(&allowed).Error
	return allowed, err
}

func (r *Repository) Effective(ctx context.Context, userID int64) ([]string, []string, error) {
	var rows []struct {
		Role       string
		Permission *string
	}
	err := r.db.WithContext(ctx).Raw(`SELECT r.code AS role, p.code AS permission FROM roles r JOIN user_roles ur ON ur.role_id = r.id JOIN users u ON u.id = ur.user_id LEFT JOIN role_permissions rp ON rp.role_id = r.id LEFT JOIN permissions p ON p.id = rp.permission_id WHERE u.id = ? AND u.status = 'active' AND r.status = 'active' ORDER BY r.code, p.code`, userID).Scan(&rows).Error
	if err != nil {
		return nil, nil, err
	}
	roles := []string{}
	permissions := []string{}
	seenRoles := map[string]bool{}
	seenPermissions := map[string]bool{}
	for _, row := range rows {
		if !seenRoles[row.Role] {
			roles = append(roles, row.Role)
			seenRoles[row.Role] = true
		}
		if row.Permission != nil && !seenPermissions[*row.Permission] {
			permissions = append(permissions, *row.Permission)
			seenPermissions[*row.Permission] = true
		}
	}
	sort.Strings(permissions)
	return roles, permissions, nil
}

func (r *Repository) ListRoles(ctx context.Context) ([]Role, error) {
	roles := []Role{}
	err := r.db.WithContext(ctx).Order("code").Find(&roles).Error
	return roles, err
}
func (r *Repository) ListPermissions(ctx context.Context) ([]Permission, error) {
	permissions := []Permission{}
	err := r.db.WithContext(ctx).Order("code").Find(&permissions).Error
	return permissions, err
}
func (r *Repository) UserRoles(ctx context.Context, userID int64) ([]Role, error) {
	var exists bool
	if err := r.db.WithContext(ctx).Raw("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", userID).Scan(&exists).Error; err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrUserNotFound
	}
	roles := []Role{}
	err := r.db.WithContext(ctx).Table("roles").Select("roles.*").Joins("JOIN user_roles ON user_roles.role_id = roles.id").Where("user_roles.user_id = ?", userID).Order("roles.code").Scan(&roles).Error
	return roles, err
}

func (r *Repository) ReplaceRoles(ctx context.Context, actorID, userID int64, roleIDs []int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", int64(5640010231148)).Error; err != nil {
			return err
		}
		var lockedID int64
		if err := tx.Raw("SELECT id FROM users WHERE id = ? FOR UPDATE", userID).Scan(&lockedID).Error; err != nil {
			return err
		}
		if lockedID == 0 {
			return ErrUserNotFound
		}
		if len(roleIDs) > 0 {
			var count int64
			if err := tx.Model(&Role{}).Where("id IN ? AND status = ?", roleIDs, "active").Count(&count).Error; err != nil {
				return err
			}
			if count != int64(len(roleIDs)) {
				return ErrInvalidRole
			}
		}
		var targetIsAdmin bool
		if err := tx.Raw(`SELECT EXISTS(SELECT 1 FROM user_roles ur JOIN roles r ON r.id=ur.role_id WHERE ur.user_id=? AND r.code='super_admin' AND r.status='active')`, userID).Scan(&targetIsAdmin).Error; err != nil {
			return err
		}
		willBeAdmin := false
		if len(roleIDs) > 0 {
			if err := tx.Raw(`SELECT EXISTS(SELECT 1 FROM roles WHERE id IN ? AND code='super_admin' AND status='active')`, roleIDs).Scan(&willBeAdmin).Error; err != nil {
				return err
			}
		}
		if targetIsAdmin && !willBeAdmin {
			var count int64
			if err := tx.Raw(`SELECT count(*) FROM user_roles ur JOIN roles r ON r.id=ur.role_id JOIN users u ON u.id=ur.user_id WHERE r.code='super_admin' AND r.status='active' AND u.status='active'`).Scan(&count).Error; err != nil {
				return err
			}
			if count <= 1 {
				return ErrLastSuperAdmin
			}
		}
		previous := []int64{}
		if err := tx.Table("user_roles").Where("user_id = ?", userID).Order("role_id").Pluck("role_id", &previous).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM user_roles WHERE user_id = ?", userID).Error; err != nil {
			return err
		}
		for _, id := range roleIDs {
			if err := tx.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", userID, id).Error; err != nil {
				return err
			}
		}
		before, err := json.Marshal(previous)
		if err != nil {
			return err
		}
		after, err := json.Marshal(roleIDs)
		if err != nil {
			return err
		}
		if err := tx.Exec("INSERT INTO rbac_audit_events (actor_user_id, target_user_id, action, previous_role_ids, new_role_ids) VALUES (?, ?, ?, ?::jsonb, ?::jsonb)", actorID, userID, "user_roles.replace", string(before), string(after)).Error; err != nil {
			return err
		}
		return tx.Exec("INSERT INTO auth_audit_events (user_id, event_type, outcome, metadata) VALUES (?, ?, ?, ?::jsonb)", actorID, "user_roles_changed", "success", fmt.Sprintf(`{"target_user_id":%d,"previous_role_ids":%s,"new_role_ids":%s}`, userID, string(before), string(after))).Error
	})
}

func (r *Repository) SetUserStatus(ctx context.Context, userID int64, status string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", int64(5640010231148)).Error; err != nil {
			return err
		}
		var isAdmin bool
		if err := tx.Raw(`SELECT EXISTS(SELECT 1 FROM users u JOIN user_roles ur ON ur.user_id=u.id JOIN roles r ON r.id=ur.role_id WHERE u.id=? AND u.status='active' AND r.code='super_admin' AND r.status='active')`, userID).Scan(&isAdmin).Error; err != nil {
			return err
		}
		if isAdmin && status != "active" {
			var count int64
			if err := tx.Raw(`SELECT count(*) FROM users u JOIN user_roles ur ON ur.user_id=u.id JOIN roles r ON r.id=ur.role_id WHERE u.status='active' AND r.code='super_admin' AND r.status='active'`).Scan(&count).Error; err != nil {
				return err
			}
			if count <= 1 {
				return ErrLastSuperAdmin
			}
		}
		return tx.Model(&authUserPlaceholder{}).Where("id=?", userID).Update("status", status).Error
	})
}

type authUserPlaceholder struct{}

func (authUserPlaceholder) TableName() string { return "users" }
func (r *Repository) SetRoleStatus(ctx context.Context, roleID int64, status string) error {
	return r.changeRole(ctx, roleID, status, false)
}
func (r *Repository) DeleteRole(ctx context.Context, roleID int64) error {
	return r.changeRole(ctx, roleID, "", true)
}
func (r *Repository) changeRole(ctx context.Context, roleID int64, status string, remove bool) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", int64(5640010231148)).Error; err != nil {
			return err
		}
		var code string
		if err := tx.Raw("SELECT code FROM roles WHERE id=? FOR UPDATE", roleID).Scan(&code).Error; err != nil {
			return err
		}
		if code == "super_admin" && (remove || status != "active") {
			var count int64
			if err := tx.Raw(`SELECT count(*) FROM users u JOIN user_roles ur ON ur.user_id=u.id JOIN roles r ON r.id=ur.role_id WHERE u.status='active' AND r.code='super_admin' AND r.status='active'`).Scan(&count).Error; err != nil {
				return err
			}
			if count <= 1 {
				return ErrLastSuperAdmin
			}
		}
		if remove {
			return tx.Exec("DELETE FROM roles WHERE id=?", roleID).Error
		}
		return tx.Exec("UPDATE roles SET status=?,updated_at=now() WHERE id=?", status, roleID).Error
	})
}
