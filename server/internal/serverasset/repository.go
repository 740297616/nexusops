package serverasset

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrNotFound       = errors.New("server not found")
	ErrTokenInvalid   = errors.New("invalid registration token")
	ErrTokenUsed      = errors.New("registration token already used")
	ErrCredential     = errors.New("agent credential unavailable")
	ErrDisabled       = errors.New("server disabled")
	ErrAlreadyAgent   = errors.New("server already has an active agent")
	ErrInvalidRequest = errors.New("invalid server request")
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil { return "", err }
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}
func newSecret() (string, []byte, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil { return "", nil, err }
	t := base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(t))
	return t, h[:], nil
}
func hashSecret(value string) []byte { h := sha256.Sum256([]byte(value)); return h[:] }

func (r *Repository) Create(ctx context.Context, s *Server) error {
	if s.ID == "" { id, err := newID(); if err != nil { return err }; s.ID = id }
	if strings.TrimSpace(s.Name) == "" || len([]rune(s.Name)) > 128 { return ErrInvalidRequest }
	s.Status, s.AgentStatus = StatusPending, "unregistered"
	return r.db.WithContext(ctx).Create(s).Error
}
func (r *Repository) List(ctx context.Context, page, size int, q, status string) ([]Server, int64, error) {
	if page < 1 || size < 1 || size > 100 { return nil, 0, ErrInvalidRequest }
	db := r.db.WithContext(ctx).Model(&Server{})
	if q != "" { like := "%" + strings.ToLower(strings.TrimSpace(q)) + "%"; db = db.Where("lower(name) LIKE ? OR lower(hostname) LIKE ?", like, like) }
	if status != "" { db = db.Where("status = ?", status) }
	var total int64
	if err := db.Count(&total).Error; err != nil { return nil, 0, err }
	var items []Server
	if err := db.Order("created_at DESC").Offset((page-1)*size).Limit(size).Find(&items).Error; err != nil { return nil, 0, err }
	return items, total, nil
}
func (r *Repository) Get(ctx context.Context, id string) (Server, error) { var s Server; err := r.db.WithContext(ctx).First(&s, "id = ?", id).Error; if errors.Is(err, gorm.ErrRecordNotFound) { return s, ErrNotFound }; return s, err }
func (r *Repository) Update(ctx context.Context, id, name, description string) error {
	if strings.TrimSpace(name) == "" || len([]rune(name)) > 128 { return ErrInvalidRequest }
	res := r.db.WithContext(ctx).Model(&Server{}).Where("id = ?", id).Updates(map[string]any{"name": name, "description": description, "updated_at": time.Now().UTC()})
	if res.Error != nil { return res.Error }; if res.RowsAffected != 1 { return ErrNotFound }; return nil
}
func (r *Repository) SetStatus(ctx context.Context, id, status string) error {
	if status != StatusPending && status != StatusOnline && status != StatusOffline && status != StatusDisabled { return ErrInvalidRequest }
	res := r.db.WithContext(ctx).Model(&Server{}).Where("id = ?", id).Updates(map[string]any{"status": status, "updated_at": time.Now().UTC()})
	if res.Error != nil { return res.Error }; if res.RowsAffected != 1 { return ErrNotFound }; return nil
}
func (r *Repository) Delete(ctx context.Context, id string) error { res := r.db.WithContext(ctx).Delete(&Server{}, "id = ?", id); if res.Error != nil { return res.Error }; if res.RowsAffected != 1 { return ErrNotFound }; return nil }

func (r *Repository) CreateRegistrationToken(ctx context.Context, serverID string, actorID int64, expiry time.Duration) (string, time.Time, error) {
	plain, hash, err := newSecret(); if err != nil { return "", time.Time{}, err }
	expires := time.Now().UTC().Add(expiry)
	return plain, expires, r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var s Server; if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&s, "id = ?", serverID).Error; err != nil { if errors.Is(err, gorm.ErrRecordNotFound) { return ErrNotFound }; return err }
		if s.Status == StatusDisabled { return ErrDisabled }
		return tx.Exec("INSERT INTO agent_registration_tokens(server_id,token_hash,expires_at,created_by) VALUES (?,?,?,?)", serverID, hash, expires, actorID).Error
	})
}

func (r *Repository) Register(ctx context.Context, token string, now time.Time) (Server, string, error) {
	hash := hashSecret(token)
	plain, credHash, err := newSecret(); if err != nil { return Server{}, "", err }
	var result Server
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rt struct { ID int64; ServerID string; ExpiresAt time.Time; UsedAt *time.Time }
		if err := tx.Raw("SELECT id,server_id,expires_at,used_at FROM agent_registration_tokens WHERE token_hash = ? FOR UPDATE", hash).Scan(&rt).Error; err != nil { return err }
		if rt.ID == 0 { return ErrTokenInvalid }; if rt.UsedAt != nil { return ErrTokenUsed }; if !rt.ExpiresAt.After(now) { return ErrTokenInvalid }
		if err := tx.Exec("UPDATE agent_registration_tokens SET used_at = ? WHERE id = ?", now, rt.ID).Error; err != nil { return err }
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&result, "id = ?", rt.ServerID).Error; err != nil { return err }
		if result.Status == StatusDisabled { return ErrDisabled }
		var count int64; if err := tx.Table("agent_credentials").Where("server_id = ? AND status = 'active'", rt.ServerID).Count(&count).Error; err != nil { return err }; if count > 0 { return ErrAlreadyAgent }
		if err := tx.Exec("INSERT INTO agent_credentials(server_id,credential_hash,status) VALUES (?,?, 'active') ON CONFLICT (server_id) DO UPDATE SET credential_hash=EXCLUDED.credential_hash,status='active',issued_at=now(),rotated_at=now(),revoked_at=NULL", rt.ServerID, credHash).Error; err != nil { return err }
		return tx.Exec("UPDATE servers SET agent_status='online',status='online',updated_at=? WHERE id=? AND status <> 'disabled'", now, rt.ServerID).Error
	})
	if err != nil { return Server{}, "", err }
	return result, plain, nil
}

func (r *Repository) Credential(ctx context.Context, credential string) (AgentPrincipal, error) {
	var p AgentPrincipal
	err := r.db.WithContext(ctx).Table("agent_credentials").Select("server_id, id AS credential_id").Where("credential_hash = ? AND status = 'active'", hashSecret(credential)).Scan(&p).Error
	if err != nil { return p, err }; if p.ServerID == "" { return p, ErrCredential }
	var status string; if err := r.db.WithContext(ctx).Table("servers").Select("status").Where("id = ?", p.ServerID).Scan(&status).Error; err != nil { return p, err }; if status == StatusDisabled { return p, ErrDisabled }; return p, nil
}
func (r *Repository) Heartbeat(ctx context.Context, serverID, version string, now time.Time) error {
	res := r.db.WithContext(ctx).Model(&Server{}).Where("id = ? AND status <> ?", serverID, StatusDisabled).Updates(map[string]any{"last_heartbeat_at": now, "agent_version": version, "agent_status": "online", "status": StatusOnline, "updated_at": now})
	if res.Error != nil { return res.Error }; if res.RowsAffected != 1 { return ErrDisabled }; return nil
}
func (r *Repository) Report(ctx context.Context, serverID string, fields map[string]any, version string, now time.Time) error {
	fields["last_report_at"], fields["last_heartbeat_at"], fields["agent_version"], fields["agent_status"], fields["status"], fields["updated_at"] = now, now, version, "online", StatusOnline, now
	res := r.db.WithContext(ctx).Model(&Server{}).Where("id = ? AND status <> ?", serverID, StatusDisabled).Updates(fields)
	if res.Error != nil { return res.Error }; if res.RowsAffected != 1 { return ErrDisabled }; return nil
}
func (r *Repository) AgentInfo(ctx context.Context, serverID string) (AgentCredentialInfo, error) { var info AgentCredentialInfo; err := r.db.WithContext(ctx).Table("agent_credentials").Where("server_id = ?", serverID).Scan(&info).Error; return info, err }
func (r *Repository) Rotate(ctx context.Context, serverID string) (string, error) { plain, hash, err := newSecret(); if err != nil { return "", err }; now := time.Now().UTC(); res := r.db.WithContext(ctx).Table("agent_credentials").Where("server_id = ? AND status='active'", serverID).Updates(map[string]any{"credential_hash": hash, "rotated_at": now}); if res.Error != nil { return "", res.Error }; if res.RowsAffected != 1 { return "", ErrCredential }; return plain, nil }
func (r *Repository) Revoke(ctx context.Context, serverID string) error { now := time.Now().UTC(); return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { res := tx.Table("agent_credentials").Where("server_id = ? AND status='active'", serverID).Updates(map[string]any{"status": "revoked", "revoked_at": now}); if res.Error != nil { return res.Error }; if res.RowsAffected == 0 { return ErrCredential }; return tx.Model(&Server{}).Where("id = ?", serverID).Updates(map[string]any{"agent_status": "revoked", "status": StatusOffline, "updated_at": now}).Error }) }
func (r *Repository) MarkOffline(ctx context.Context, cutoff time.Time) error { return r.db.WithContext(ctx).Exec("UPDATE servers SET status='offline', agent_status='offline', updated_at=now() WHERE status='online' AND last_heartbeat_at IS NOT NULL AND last_heartbeat_at < ?", cutoff).Error }

