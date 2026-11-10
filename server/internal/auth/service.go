package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

const AccessTTL = 15 * time.Minute
const RefreshTTL = 7 * 24 * time.Hour

var usernamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{2,63}$`)

type Tokens struct {
	AccessToken      string    `json:"access_token"`
	AccessExpiresAt  time.Time `json:"access_expires_at"`
	RefreshToken     string    `json:"refresh_token"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
	TokenType        string    `json:"token_type"`
}

type Service struct {
	repo      *Repository
	dummyHash string
	now       func() time.Time
}

func NewService(repo *Repository) (*Service, error) {
	dummy, err := newToken()
	if err != nil {
		return nil, err
	}
	hash, err := HashPassword(dummy)
	if err != nil {
		return nil, err
	}
	return &Service{repo: repo, dummyHash: hash, now: time.Now}, nil
}

func NormalizeUsername(username string) (string, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if !usernamePattern.MatchString(username) {
		return "", errors.New("username must be 3-64 ASCII letters, digits, underscores, hyphens or dots")
	}
	return username, nil
}

func (s *Service) InitializeAdmin(ctx context.Context, username, display, password string) error {
	username, err := NormalizeUsername(username)
	if err != nil {
		return err
	}
	display = strings.TrimSpace(display)
	if display == "" || len([]rune(display)) > 128 {
		return errors.New("display name must be 1-128 characters")
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	return s.repo.InitializeAdmin(ctx, username, display, hash)
}

func (s *Service) Login(ctx context.Context, username, password string) (Tokens, error) {
	name, validationErr := NormalizeUsername(username)
	var user User
	var err error
	if validationErr == nil {
		user, err = s.repo.FindUser(ctx, name)
	} else {
		err = gorm.ErrRecordNotFound
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return Tokens{}, err
	}
	hash := s.dummyHash
	if err == nil {
		hash = user.PasswordHash
	}
	valid := VerifyPassword(hash, password)
	if !valid || err != nil || user.Status != "active" {
		return Tokens{}, ErrInvalid
	}
	access, err := newToken()
	if err != nil {
		return Tokens{}, err
	}
	refresh, err := newToken()
	if err != nil {
		return Tokens{}, err
	}
	now := s.now().UTC()
	expires := now.Add(RefreshTTL)
	accessExpires := now.Add(AccessTTL)
	session := Session{UserID: user.ID, AccessHash: hashToken(access), AccessExpiresAt: accessExpires, ExpiresAt: expires}
	refreshRecord := RefreshToken{TokenHash: hashToken(refresh), ExpiresAt: expires}
	if err := s.repo.CreateSession(ctx, user, &session, &refreshRecord, now); err != nil {
		return Tokens{}, err
	}
	return Tokens{access, accessExpires, refresh, expires, "Bearer"}, nil
}

func (s *Service) Refresh(ctx context.Context, token string) (Tokens, error) {
	if !validToken(token) {
		return Tokens{}, ErrInvalid
	}
	access, err := newToken()
	if err != nil {
		return Tokens{}, err
	}
	refresh, err := newToken()
	if err != nil {
		return Tokens{}, err
	}
	accessExpires, refreshExpires, err := s.repo.Rotate(ctx, hashToken(token), hashToken(access), hashToken(refresh), s.now().UTC(), AccessTTL)
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{access, accessExpires, refresh, refreshExpires, "Bearer"}, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (User, int64, error) {
	if !validToken(token) {
		return User{}, 0, ErrInvalid
	}
	return s.repo.Authenticate(ctx, hashToken(token), s.now().UTC())
}
func (s *Service) Logout(ctx context.Context, sessionID int64) error {
	return s.repo.Revoke(ctx, sessionID, s.now().UTC())
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func validToken(value string) bool {
	if len(value) != 43 {
		return false
	}
	b, err := base64.RawURLEncoding.Strict().DecodeString(value)
	return err == nil && len(b) == 32
}
func hashToken(token string) []byte { sum := sha256.Sum256([]byte(token)); return sum[:] }
