package auth

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInvalid = errors.New("invalid credentials")
var ErrAdminExists = errors.New("an administrator already exists")

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) FindUser(ctx context.Context, username string) (User, error) {
	var user User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	return user, err
}

func (r *Repository) InitializeAdmin(ctx context.Context, username, display, hash string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", int64(5640010231147)).Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&User{}).Where("role = ?", "admin").Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrAdminExists
		}
		user := User{Username: username, DisplayName: display, PasswordHash: hash, Status: "active", Role: "admin"}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		result := tx.Exec("INSERT INTO user_roles (user_id, role_id) SELECT ?, id FROM roles WHERE code = ? AND status = 'active'", user.ID, "super_admin")
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("active super_admin role is unavailable; apply migrations")
		}
		return nil
	})
}

func (r *Repository) CreateSession(ctx context.Context, user User, session *Session, refresh *RefreshToken, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, user.ID).Error; err != nil {
			return err
		}
		if current.Status != "active" || current.PasswordHash != user.PasswordHash {
			return ErrInvalid
		}
		if err := tx.Create(session).Error; err != nil {
			return err
		}
		refresh.SessionID = session.ID
		if err := tx.Create(refresh).Error; err != nil {
			return err
		}
		return tx.Model(&current).Updates(map[string]any{"last_login_at": now, "updated_at": now}).Error
	})
}

func (r *Repository) Authenticate(ctx context.Context, hash []byte, now time.Time) (User, int64, error) {
	var session Session
	err := r.db.WithContext(ctx).Where("access_hash = ? AND revoked_at IS NULL AND access_expires_at > ? AND expires_at > ?", hash, now, now).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, 0, ErrInvalid
	}
	if err != nil {
		return User{}, 0, err
	}
	var user User
	err = r.db.WithContext(ctx).Where("id = ? AND status = ?", session.UserID, "active").First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, 0, ErrInvalid
	}
	return user, session.ID, err
}

func (r *Repository) Rotate(ctx context.Context, oldHash, accessHash, nextHash []byte, now time.Time, accessTTL time.Duration) (time.Time, time.Time, error) {
	var accessExpires, refreshExpires time.Time
	replayed := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var old RefreshToken
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("token_hash = ?", oldHash).First(&old).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvalid
			}
			return err
		}
		var session Session
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&session, old.SessionID).Error; err != nil {
			return err
		}
		if session.RevokedAt != nil || !session.ExpiresAt.After(now) || !old.ExpiresAt.After(now) {
			return ErrInvalid
		}
		if old.ConsumedAt != nil {
			replayed = true
			return tx.Model(&session).Update("revoked_at", now).Error
		}
		var user User
		if err := tx.Where("id = ? AND status = ?", session.UserID, "active").First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvalid
			}
			return err
		}
		accessExpires = now.Add(accessTTL)
		if accessExpires.After(session.ExpiresAt) {
			accessExpires = session.ExpiresAt
		}
		refreshExpires = session.ExpiresAt
		if err := tx.Model(&old).Update("consumed_at", now).Error; err != nil {
			return err
		}
		if err := tx.Model(&session).Updates(map[string]any{"access_hash": accessHash, "access_expires_at": accessExpires}).Error; err != nil {
			return err
		}
		return tx.Create(&RefreshToken{SessionID: session.ID, TokenHash: nextHash, ExpiresAt: refreshExpires}).Error
	})
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if replayed {
		return time.Time{}, time.Time{}, ErrRefreshReplay
	}
	return accessExpires, refreshExpires, nil
}

func (r *Repository) Revoke(ctx context.Context, sessionID int64, now time.Time) error {
	return r.db.WithContext(ctx).Model(&Session{}).Where("id = ? AND revoked_at IS NULL", sessionID).Update("revoked_at", now).Error
}

func (r *Repository) ListSessions(ctx context.Context, userID, currentID int64, now time.Time) ([]SessionInfo, error) {
	items := []SessionInfo{}
	err := r.db.WithContext(ctx).Raw(`SELECT id, created_at, access_expires_at, expires_at, (id = ?) AS current FROM auth_sessions WHERE user_id = ? AND revoked_at IS NULL AND expires_at > ? ORDER BY created_at DESC`, currentID, userID, now).Scan(&items).Error
	return items, err
}
func (r *Repository) RevokeSession(ctx context.Context, userID, sessionID int64, now time.Time) error {
	return r.db.WithContext(ctx).Model(&Session{}).Where("id = ? AND user_id = ? AND revoked_at IS NULL", sessionID, userID).Update("revoked_at", now).Error
}
func (r *Repository) RevokeOtherSessions(ctx context.Context, userID, currentID int64, now time.Time) error {
	return r.db.WithContext(ctx).Model(&Session{}).Where("user_id = ? AND id <> ? AND revoked_at IS NULL", userID, currentID).Update("revoked_at", now).Error
}
func (r *Repository) Cleanup(ctx context.Context, now time.Time, sessionRetention, auditRetention time.Duration, batch int) (int64, error) {
	var removed int64
	cutoff := now.Add(-sessionRetention)
	for {
		result := r.db.WithContext(ctx).Exec(`DELETE FROM auth_refresh_tokens WHERE id IN (SELECT id FROM auth_refresh_tokens WHERE (expires_at < ? OR consumed_at < ?) ORDER BY id LIMIT ?)`, now, cutoff, batch)
		if result.Error != nil {
			return removed, result.Error
		}
		removed += result.RowsAffected
		if result.RowsAffected < int64(batch) {
			break
		}
	}
	for {
		result := r.db.WithContext(ctx).Exec(`DELETE FROM auth_sessions WHERE id IN (SELECT id FROM auth_sessions WHERE ((expires_at < ?) OR (revoked_at IS NOT NULL AND revoked_at < ?)) ORDER BY id LIMIT ?)`, now, cutoff, batch)
		if result.Error != nil {
			return removed, result.Error
		}
		removed += result.RowsAffected
		if result.RowsAffected < int64(batch) {
			break
		}
	}
	if auditRetention > 0 {
		for {
			result := r.db.WithContext(ctx).Exec(`DELETE FROM auth_audit_events WHERE id IN (SELECT id FROM auth_audit_events WHERE created_at < ? ORDER BY id LIMIT ?)`, now.Add(-auditRetention), batch)
			if result.Error != nil {
				return removed, result.Error
			}
			removed += result.RowsAffected
			if result.RowsAffected < int64(batch) {
				break
			}
		}
	}
	// Agent rate-limit windows are ephemeral; prune old keys in bounded batches.
	for {
		result := r.db.WithContext(ctx).Exec(`DELETE FROM agent_rate_limits WHERE rate_key IN (SELECT rate_key FROM agent_rate_limits WHERE updated_at < ? ORDER BY updated_at LIMIT ?)`, now.Add(-24*time.Hour), batch)
		if result.Error != nil {
			// Older databases may not have the Phase 3.0.5 table yet.
			break
		}
		removed += result.RowsAffected
		if result.RowsAffected < int64(batch) { break }
	}
	return removed, nil
}

func (r *Repository) AllowLogin(ctx context.Context, ipKey, accountKey string, now time.Time, window time.Duration, ipMax, accountMax int) (bool, error) {
	allowed := true
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range []struct {
			key string
			max int
		}{{ipKey, ipMax}, {accountKey, accountMax}} {
			var within bool
			if err := tx.Raw(`INSERT INTO auth_rate_limits (rate_key, window_started_at, attempts, updated_at) VALUES (?, ?, 1, ?) ON CONFLICT (rate_key) DO UPDATE SET window_started_at = CASE WHEN auth_rate_limits.window_started_at <= ? THEN EXCLUDED.window_started_at ELSE auth_rate_limits.window_started_at END, attempts = CASE WHEN auth_rate_limits.window_started_at <= ? THEN 1 ELSE auth_rate_limits.attempts + 1 END, updated_at = EXCLUDED.updated_at RETURNING attempts <= ?`, item.key, now, now, now.Add(-window), now.Add(-window), item.max).Scan(&within).Error; err != nil {
				return err
			}
			if !within {
				allowed = false
				return nil
			}
		}
		return nil
	})
	return allowed, err
}
