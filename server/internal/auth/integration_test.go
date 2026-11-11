package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/740297616/nexusops/server/internal/api"
	"github.com/740297616/nexusops/server/internal/config"
	"github.com/740297616/nexusops/server/internal/database"
	"github.com/gin-gonic/gin"
)

// Requires a separate, empty disposable database; migrations modify its schema.
func TestAuthenticationPostgres(t *testing.T) {
	name := os.Getenv("NEXUSOPS_AUTH_TEST_DB_NAME")
	if os.Getenv("NEXUSOPS_DB_INTEGRATION") != "1" || name == "" {
		t.Skip("set NEXUSOPS_DB_INTEGRATION=1 and NEXUSOPS_AUTH_TEST_DB_NAME to a separate empty test database")
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Database.Name = name
	if name == os.Getenv("DB_NAME") {
		t.Fatal("authentication test database must differ from DB_NAME")
	}
	db, err := database.Open(context.Background(), cfg.Database)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	provider, err := db.Migrations()
	if err != nil {
		t.Fatal(err)
	}
	statuses, err := provider.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, status := range statuses {
		if string(status.State) != "pending" {
			t.Fatal("authentication test database must be empty")
		}
	}
	if _, err := provider.Up(context.Background()); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(NewRepository(db.GORM()))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	const password = "isolated test password"
	if err := service.InitializeAdmin(ctx, "Admin", "Test admin", password); err != nil {
		t.Fatal(err)
	}
	if err := service.InitializeAdmin(ctx, "Other", "Another admin", password); !errors.Is(err, ErrAdminExists) {
		t.Fatalf("duplicate init: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := api.NewRouter("test", db.Ping)
	RegisterRoutes(router, service, func(context.Context, int64) ([]string, []string, error) { return []string{}, []string{}, nil })
	var logs bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(previous)
	call := func(method, path, body, token string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	login := func(username, pass string) Tokens {
		t.Helper()
		payload, _ := json.Marshal(map[string]string{"username": username, "password": pass})
		w := call(http.MethodPost, "/api/v1/auth/login", string(payload), "")
		if w.Code != 200 {
			t.Fatalf("login status %d: %s", w.Code, w.Body.String())
		}
		var response struct {
			Data Tokens `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		return response.Data
	}
	for _, credentials := range []string{`{"username":"admin","password":"incorrect test password"}`, `{"username":"missing","password":"incorrect test password"}`} {
		w := call("POST", "/api/v1/auth/login", credentials, "")
		if w.Code != 401 {
			t.Fatalf("bad login: %d", w.Code)
		}
	}
	if w := call("GET", "/api/v1/auth/me", "", ""); w.Code != 401 {
		t.Fatalf("missing token: %d", w.Code)
	}
	tokens := login("ADMIN", password)
	if w := call("GET", "/api/v1/auth/me", "", tokens.AccessToken); w.Code != 200 || strings.Contains(w.Body.String(), "password_hash") || strings.Contains(w.Body.String(), password) {
		t.Fatalf("me: %d %s", w.Code, w.Body.String())
	}
	var count int64
	if err := db.GORM().Raw("SELECT count(*) FROM auth_refresh_tokens WHERE token_hash = ?", []byte(tokens.RefreshToken)).Scan(&count).Error; err != nil || count != 0 {
		t.Fatal("plaintext refresh token stored")
	}
	refreshBody := func(token string) string {
		payload, _ := json.Marshal(map[string]string{"refresh_token": token})
		return string(payload)
	}
	result := call("POST", "/api/v1/auth/refresh", refreshBody(tokens.RefreshToken), "")
	if result.Code != 200 {
		t.Fatalf("refresh: %d %s", result.Code, result.Body.String())
	}
	var rotated struct {
		Data Tokens `json:"data"`
	}
	if err := json.Unmarshal(result.Body.Bytes(), &rotated); err != nil {
		t.Fatal(err)
	}
	if rotated.Data.RefreshToken == tokens.RefreshToken || rotated.Data.AccessToken == tokens.AccessToken {
		t.Fatal("tokens did not rotate")
	}
	if w := call("GET", "/api/v1/auth/me", "", tokens.AccessToken); w.Code != 401 {
		t.Fatal("old access token remains active")
	}
	if w := call("POST", "/api/v1/auth/refresh", refreshBody(tokens.RefreshToken), ""); w.Code != 401 {
		t.Fatal("replayed refresh accepted")
	}
	if w := call("GET", "/api/v1/auth/me", "", rotated.Data.AccessToken); w.Code != 401 {
		t.Fatal("replay did not revoke access")
	}
	if w := call("POST", "/api/v1/auth/refresh", refreshBody(rotated.Data.RefreshToken), ""); w.Code != 401 {
		t.Fatal("replay did not revoke session")
	}

	tokens = login("admin", password)
	if w := call("POST", "/api/v1/auth/logout", "", tokens.AccessToken); w.Code != 200 {
		t.Fatal("logout failed")
	}
	if w := call("GET", "/api/v1/auth/me", "", tokens.AccessToken); w.Code != 401 {
		t.Fatal("logout did not revoke access")
	}
	if w := call("POST", "/api/v1/auth/refresh", refreshBody(tokens.RefreshToken), ""); w.Code != 401 {
		t.Fatal("logout did not revoke refresh")
	}

	tokens = login("admin", password)
	now := time.Now().UTC()
	service.now = func() time.Time { return now.Add(AccessTTL + time.Second) }
	if w := call("GET", "/api/v1/auth/me", "", tokens.AccessToken); w.Code != 401 {
		t.Fatal("expired access accepted")
	}
	service.now = func() time.Time { return now.Add(RefreshTTL + time.Second) }
	if w := call("POST", "/api/v1/auth/refresh", refreshBody(tokens.RefreshToken), ""); w.Code != 401 {
		t.Fatal("expired refresh accepted")
	}
	service.now = time.Now

	tokens = login("admin", password)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	successful := make(chan Tokens, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			next, err := service.Refresh(ctx, tokens.RefreshToken)
			results <- err
			if err == nil {
				successful <- next
			}
		}()
	}
	wg.Wait()
	close(results)
	close(successful)
	successes, rejected := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, ErrInvalid) {
			rejected++
		} else {
			t.Fatal(err)
		}
	}
	if successes != 1 || rejected != 1 {
		t.Fatalf("concurrent refresh: successes=%d rejected=%d", successes, rejected)
	}
	for next := range successful {
		if _, _, err := service.Authenticate(ctx, next.AccessToken); !errors.Is(err, ErrInvalid) {
			t.Fatal("concurrent replay did not revoke session")
		}
	}
	if strings.Contains(logs.String(), password) || strings.Contains(logs.String(), tokens.AccessToken) || strings.Contains(logs.String(), tokens.RefreshToken) {
		t.Fatal("credentials leaked to request logs")
	}
	for _, path := range []string{"/api/v1/health", "/api/v1/ready"} {
		if w := call("GET", path, "", ""); w.Code != 200 {
			t.Fatalf("health regression: %s %d", path, w.Code)
		}
	}
}
