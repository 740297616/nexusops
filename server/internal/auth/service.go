package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

const AccessTTL = 15 * time.Minute
const RefreshTTL = 7 * 24 * time.Hour

var usernamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{2,63}$`)
var ErrRateLimited = errors.New("authentication rate limited")
var ErrRefreshReplay = fmt.Errorf("%w: refresh token replay detected", ErrInvalid)

type Options struct {
	TrustedProxyCIDRs                   []string
	RateLimitIPMax, RateLimitAccountMax int
	RateLimitWindow                     time.Duration
}

type Tokens struct {
	AccessToken      string    `json:"access_token"`
	AccessExpiresAt  time.Time `json:"access_expires_at"`
	RefreshToken     string    `json:"refresh_token"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
	TokenType        string    `json:"token_type"`
}

type Service struct {
	repo                                *Repository
	dummyHash                           string
	now                                 func() time.Time
	trustedProxies                      []*net.IPNet
	rateLimitIPMax, rateLimitAccountMax int
	rateLimitWindow                     time.Duration
}

func NewService(repo *Repository) (*Service, error) {
	return NewServiceWithOptions(repo, Options{RateLimitIPMax: 20, RateLimitAccountMax: 10, RateLimitWindow: time.Minute})
}
func NewServiceWithOptions(repo *Repository, options Options) (*Service, error) {
	dummy, err := newToken()
	if err != nil {
		return nil, err
	}
	hash, err := HashPassword(dummy)
	if err != nil {
		return nil, err
	}
	if options.RateLimitIPMax < 1 {
		options.RateLimitIPMax = 20
	}
	if options.RateLimitAccountMax < 1 {
		options.RateLimitAccountMax = 10
	}
	if options.RateLimitWindow <= 0 {
		options.RateLimitWindow = time.Minute
	}
	proxies := []*net.IPNet{}
	for _, cidr := range options.TrustedProxyCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, errors.New("invalid trusted proxy CIDR")
		}
		proxies = append(proxies, network)
	}
	return &Service{repo: repo, dummyHash: hash, now: time.Now, trustedProxies: proxies, rateLimitIPMax: options.RateLimitIPMax, rateLimitAccountMax: options.RateLimitAccountMax, rateLimitWindow: options.RateLimitWindow}, nil
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

func (s *Service) AllowLogin(ctx context.Context, sourceIP, username string) (bool, error) {
	name, err := NormalizeUsername(username)
	accountKey := "account:unknown"
	if err == nil {
		if _, findErr := s.repo.FindUser(ctx, name); findErr == nil {
			accountKey = "account:" + name
		} else if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return false, findErr
		}
	}
	if sourceIP == "" {
		sourceIP = "unknown"
	}
	return s.repo.AllowLogin(ctx, "ip:"+sourceIP, accountKey, s.now().UTC(), s.rateLimitWindow, s.rateLimitIPMax, s.rateLimitAccountMax)
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
func (s *Service) Sessions(ctx context.Context, userID, currentID int64) ([]SessionInfo, error) {
	return s.repo.ListSessions(ctx, userID, currentID, s.now().UTC())
}
func (s *Service) RevokeSession(ctx context.Context, userID, sessionID int64) error {
	return s.repo.RevokeSession(ctx, userID, sessionID, s.now().UTC())
}
func (s *Service) RevokeOtherSessions(ctx context.Context, userID, currentID int64) error {
	return s.repo.RevokeOtherSessions(ctx, userID, currentID, s.now().UTC())
}
func (s *Service) Cleanup(ctx context.Context, sessionRetention, auditRetention time.Duration) (int64, error) {
	return s.repo.Cleanup(ctx, s.now().UTC(), sessionRetention, auditRetention, 100)
}
func (s *Service) StartCleanup(ctx context.Context, interval, sessionRetention, auditRetention time.Duration) {
	if interval <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if removed, err := s.Cleanup(ctx, sessionRetention, auditRetention); err != nil {
					log.Printf("authentication cleanup failed: %v", err)
				} else if removed > 0 {
					log.Printf("authentication cleanup removed records=%d", removed)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
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
