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
		return time.Time{}, time.Time{}, ErrInvalid
	}
	return accessExpires, refreshExpires, nil
}

func (r *Repository) Revoke(ctx context.Context, sessionID int64, now time.Time) error {
	return r.db.WithContext(ctx).Model(&Session{}).Where("id = ? AND revoked_at IS NULL", sessionID).Update("revoked_at", now).Error
}
