package rbac

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/740297616/nexusops/server/internal/api"
	"github.com/740297616/nexusops/server/internal/auth"
	"github.com/740297616/nexusops/server/internal/config"
	"github.com/740297616/nexusops/server/internal/database"
	"github.com/gin-gonic/gin"
)

// This test migrates and changes permissions; use a separate empty database.
func TestRBACPostgres(t *testing.T) {
	name := os.Getenv("NEXUSOPS_RBAC_TEST_DB_NAME")
	if os.Getenv("NEXUSOPS_DB_INTEGRATION") != "1" || name == "" {
		t.Skip("set NEXUSOPS_RBAC_TEST_DB_NAME to a separate empty database")
	}
	if name == os.Getenv("DB_NAME") || name == os.Getenv("NEXUSOPS_AUTH_TEST_DB_NAME") {
		t.Fatal("RBAC test database must be separate")
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Database.Name = name
	db, err := database.Open(context.Background(), cfg.Database)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	provider, err := db.Migrations()
	if err != nil {
		t.Fatal(err)
	}
	statuses, err := provider.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, status := range statuses {
		if string(status.State) != "pending" {
			t.Fatal("test database must be empty")
		}
	}
	if _, err := provider.UpTo(ctx, 2); err != nil {
		t.Fatal(err)
	}
	hash, err := auth.HashPassword("isolated rbac password")
	if err != nil {
		t.Fatal(err)
	}
	legacy := auth.User{Username: "legacyadmin", DisplayName: "Legacy admin", PasswordHash: hash, Status: "active", Role: "admin"}
	if err := db.GORM().Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	legacySession := auth.Session{UserID: legacy.ID, AccessHash: []byte("legacy access hash"), AccessExpiresAt: time.Now().Add(time.Hour), ExpiresAt: time.Now().Add(time.Hour)}
	if err := db.GORM().Create(&legacySession).Error; err != nil {
		t.Fatal(err)
	}
	legacyRefresh := auth.RefreshToken{SessionID: legacySession.ID, TokenHash: []byte("legacy refresh hash"), ExpiresAt: time.Now().Add(time.Hour)}
	if err := db.GORM().Create(&legacyRefresh).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	if again, err := provider.Up(ctx); err != nil || len(again) != 0 {
		t.Fatalf("repeat up: %v %v", again, err)
	}
	service := NewService(NewRepository(db.GORM()))
	if allowed, err := service.HasPermission(ctx, legacy.ID, PermissionRoleManage); err != nil || !allowed {
		t.Fatal("legacy admin not mapped")
	}
	var count int64
	if err := db.GORM().Raw("SELECT count(*) FROM auth_sessions s JOIN auth_refresh_tokens t ON t.session_id=s.id WHERE s.id=? AND t.id=?", legacySession.ID, legacyRefresh.ID).Scan(&count).Error; err != nil || count != 1 {
		t.Fatal("migration changed existing session data")
	}
	if _, err := provider.Down(ctx); err != nil {
		t.Fatal(err)
	}
	if err := db.GORM().Raw("SELECT count(*) FROM users u JOIN auth_sessions s ON s.user_id=u.id JOIN auth_refresh_tokens t ON t.session_id=s.id WHERE u.id=? AND s.id=? AND t.id=?", legacy.ID, legacySession.ID, legacyRefresh.ID).Scan(&count).Error; err != nil || count != 1 {
		t.Fatal("rollback changed auth data")
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	if again, err := provider.Up(ctx); err != nil || len(again) != 0 {
		t.Fatalf("repeat up: %v %v", again, err)
	}

	user := auth.User{Username: "operatoruser", DisplayName: "Operator", PasswordHash: hash, Status: "active", Role: "viewer"}
	if err := db.GORM().Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	var operator, admin Role
	if err := db.GORM().Where("code = ?", "operator").First(&operator).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.GORM().Where("code = ?", "admin").First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	check := func(permission string, want bool) {
		t.Helper()
		got, err := service.HasPermission(ctx, user.ID, permission)
		if err != nil || got != want {
			t.Fatalf("permission %s: got %v err %v", permission, got, err)
		}
	}
	check(PermissionRoleRead, false)
	if err := service.ReplaceRoles(ctx, legacy.ID, user.ID, []int64{operator.ID}); err != nil {
		t.Fatal(err)
	}
	check(PermissionRoleRead, true)
	check(PermissionUserRead, false)
	if err := service.ReplaceRoles(ctx, legacy.ID, user.ID, []int64{operator.ID, admin.ID}); err != nil {
		t.Fatal(err)
	}
	check(PermissionRoleRead, true)
	check(PermissionUserRead, true)
	check(PermissionPermissionRead, true)
	if err := db.GORM().Exec("DELETE FROM role_permissions WHERE role_id = ? AND permission_id = (SELECT id FROM permissions WHERE code = ?)", admin.ID, PermissionUserRead).Error; err != nil {
		t.Fatal(err)
	}
	check(PermissionUserRead, false)
	if again, err := provider.Up(ctx); err != nil || len(again) != 0 {
		t.Fatal("repeat init failed")
	}
	check(PermissionUserRead, false)
	if err := db.GORM().Exec("UPDATE roles SET status = 'disabled' WHERE id = ?", operator.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.ReplaceRoles(ctx, legacy.ID, user.ID, []int64{operator.ID}); !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("disabled role: %v", err)
	}
	if err := service.ReplaceRoles(ctx, legacy.ID, user.ID, []int64{999999}); !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("unknown role: %v", err)
	}
	if err := service.ReplaceRoles(ctx, legacy.ID, 999999, []int64{}); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("unknown user: %v", err)
	}
	if err := service.ReplaceRoles(ctx, legacy.ID, user.ID, []int64{admin.ID, admin.ID}); !errors.Is(err, ErrInvalidAssignment) {
		t.Fatalf("duplicate role: %v", err)
	}
	if allowed, err := service.HasPermission(ctx, 999999, PermissionRoleRead); err != nil || allowed {
		t.Fatal("unknown user authorized")
	}
	if allowed, err := service.HasPermission(ctx, user.ID, "missing:permission"); err != nil || allowed {
		t.Fatal("unknown permission authorized")
	}
	if err := service.ReplaceRoles(ctx, legacy.ID, user.ID, []int64{}); err != nil {
		t.Fatal(err)
	}
	check(PermissionRoleRead, false)
	if err := db.GORM().Exec("UPDATE roles SET status = 'active' WHERE id = ?", operator.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.ReplaceRoles(ctx, legacy.ID, user.ID, []int64{operator.ID}); err != nil {
		t.Fatal(err)
	}
	if err := db.GORM().Exec("UPDATE roles SET status = 'disabled' WHERE id = ?", operator.ID).Error; err != nil {
		t.Fatal(err)
	}
	check(PermissionRoleRead, false)
	if err := db.GORM().Exec("UPDATE roles SET status = 'active' WHERE id = ?", operator.ID).Error; err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	loginService, err := auth.NewService(auth.NewRepository(db.GORM()))
	if err != nil {
		t.Fatal(err)
	}
	router := api.NewRouter("test", db.Ping)
	auth.RegisterRoutes(router, loginService, service.Effective)
	RegisterRoutes(router, loginService, service)
	adminTokens, err := loginService.Login(ctx, "legacyadmin", "isolated rbac password")
	if err != nil {
		t.Fatal(err)
	}
	userTokens, err := loginService.Login(ctx, "operatoruser", "isolated rbac password")
	if err != nil {
		t.Fatal(err)
	}
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
	if w := call("GET", "/api/v1/roles", "", ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("missing auth: %d", w.Code)
	}
	if w := call("GET", "/api/v1/permissions?role=super_admin", "", userTokens.AccessToken); w.Code != http.StatusForbidden {
		t.Fatalf("forbidden: %d", w.Code)
	}
	if w := call("GET", "/api/v1/roles", "", userTokens.AccessToken); w.Code != http.StatusOK {
		t.Fatalf("authorized: %d", w.Code)
	}
	if w := call("PUT", fmt.Sprintf("/api/v1/users/%d/roles", user.ID), fmt.Sprintf(`{"role_ids":[%d]}`, admin.ID), userTokens.AccessToken); w.Code != http.StatusForbidden {
		t.Fatalf("unauthorized assignment: %d", w.Code)
	}
	if w := call("GET", "/api/v1/users/999999/roles", "", adminTokens.AccessToken); w.Code != http.StatusNotFound {
		t.Fatalf("unknown user: %d", w.Code)
	}
	if w := call("PUT", fmt.Sprintf("/api/v1/users/%d/roles", user.ID), `{"role_ids":[999999]}`, adminTokens.AccessToken); w.Code != http.StatusBadRequest {
		t.Fatalf("unknown role: %d", w.Code)
	}
	if w := call("PUT", fmt.Sprintf("/api/v1/users/%d/roles", user.ID), fmt.Sprintf(`{"role_ids":[%d]}`, admin.ID), adminTokens.AccessToken); w.Code != http.StatusOK {
		t.Fatalf("assignment: %d %s", w.Code, w.Body.String())
	}
	if w := call("GET", "/api/v1/permissions", "", userTokens.AccessToken); w.Code != http.StatusOK {
		t.Fatalf("updated permissions: %d", w.Code)
	}
	me := call("GET", "/api/v1/auth/me", "", userTokens.AccessToken)
	var profile struct {
		Data struct {
			Username    string   `json:"username"`
			Role        string   `json:"role"`
			Roles       []string `json:"roles"`
			Permissions []string `json:"permissions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(me.Body.Bytes(), &profile); err != nil || me.Code != 200 || profile.Data.Username != user.Username || profile.Data.Role != "viewer" || len(profile.Data.Roles) != 1 || profile.Data.Roles[0] != "admin" || len(profile.Data.Permissions) == 0 {
		t.Fatalf("profile: %d %s", me.Code, me.Body.String())
	}
	if err := db.GORM().Raw("SELECT count(*) FROM rbac_audit_events WHERE actor_user_id=? AND target_user_id=?", legacy.ID, user.ID).Scan(&count).Error; err != nil || count == 0 {
		t.Fatal("role changes were not audited")
	}

	var wg sync.WaitGroup
	failures := make(chan error, 32)
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := service.HasPermission(ctx, user.ID, PermissionRoleRead)
			if err != nil {
				failures <- err
			}
		}()
	}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := service.ReplaceRoles(ctx, legacy.ID, user.ID, []int64{}); err != nil {
				failures <- err
			}
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
	check(PermissionRoleRead, false)
	var readers sync.WaitGroup
	revokedFailures := make(chan error, 24)
	for i := 0; i < 24; i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			allowed, err := service.HasPermission(ctx, user.ID, PermissionRoleRead)
			if err != nil || allowed {
				revokedFailures <- fmt.Errorf("revoked permission allowed=%v err=%v", allowed, err)
			}
		}()
	}
	readers.Wait()
	close(revokedFailures)
	for err := range revokedFailures {
		t.Fatal(err)
	}
	if w := call("GET", "/api/v1/roles", "", userTokens.AccessToken); w.Code != http.StatusForbidden {
		t.Fatalf("revocation not immediate: %d", w.Code)
	}
}
