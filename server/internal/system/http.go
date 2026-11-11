package system

import (
	"encoding/json"
	"github.com/740297616/nexusops/server/internal/auth"
	"github.com/740297616/nexusops/server/internal/platform"
	"github.com/740297616/nexusops/server/internal/rbac"
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
)

func RegisterRoutes(r *gin.Engine, a *auth.Service, x *rbac.Service, s *Service) {
	g := r.Group("/api/v1")
	g.Use(auth.RequireAuthentication(a))
	h := &handler{s: s}
	g.GET("/users", rbac.RequirePermission(x, rbac.PermissionUserRead), h.users)
	g.GET("/users/:id", rbac.RequirePermission(x, rbac.PermissionUserRead), h.user)
	g.POST("/users", rbac.RequirePermission(x, rbac.PermissionUserCreate), h.create)
	g.PUT("/users/:id", rbac.RequirePermission(x, rbac.PermissionUserUpdate), h.update)
	g.PATCH("/users/:id/status", rbac.RequirePermission(x, rbac.PermissionUserUpdate), h.status)
	g.PUT("/users/:id/password", rbac.RequirePermission(x, rbac.PermissionUserUpdate), h.password)
	g.GET("/audit/auth", rbac.RequirePermission(x, rbac.PermissionAuditRead), h.authAudit)
	g.GET("/audit/roles", rbac.RequirePermission(x, rbac.PermissionAuditRead), h.roleAudit)
	g.GET("/audit/operations", rbac.RequirePermission(x, rbac.PermissionAuditRead), h.authAudit)
	g.GET("/admin/sessions", rbac.RequirePermission(x, rbac.PermissionSessionRead), h.sessions)
	g.GET("/admin/users/:id/sessions", rbac.RequirePermission(x, rbac.PermissionSessionRead), h.userSessions)
	g.DELETE("/admin/users/:id/sessions", rbac.RequirePermission(x, rbac.PermissionSessionManage), h.revokeUserSessions)
	g.GET("/system/config", rbac.RequirePermission(x, rbac.PermissionConfigRead), h.config)
	g.PUT("/system/config", rbac.RequirePermission(x, rbac.PermissionConfigUpdate), h.setConfig)
}

type handler struct{ s *Service }

func id(c *gin.Context) (int64, bool) {
	v, e := strconv.ParseInt(c.Param("id"), 10, 64)
	if e != nil || v < 1 {
		platform.Respond(c, 400, nil, &platform.ErrorBody{Code: "invalid_request", Message: "Invalid ID"})
		return 0, false
	}
	return v, true
}
func page(c *gin.Context) (int, int) {
	p, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	z, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if p < 1 {
		p = 1
	}
	if z < 1 || z > 100 {
		z = 20
	}
	return p, z
}
func out(c *gin.Context, v any) { platform.Respond(c, 200, v, nil) }
func bad(c *gin.Context, e error) {
	platform.Respond(c, 400, nil, &platform.ErrorBody{Code: "invalid_request", Message: e.Error()})
}
func (h *handler) users(c *gin.Context) {
	p, z := page(c)
	v, t, e := h.s.ListUsers(c, p, z, c.Query("q"), c.Query("status"))
	if e != nil {
		bad(c, e)
		return
	}
	out(c, Page{v, t, p, z})
}
func (h *handler) user(c *gin.Context) {
	i, ok := id(c)
	if !ok {
		return
	}
	v, e := h.s.GetUser(c, i)
	if e != nil {
		platform.Respond(c, 404, nil, &platform.ErrorBody{Code: "not_found", Message: "User not found"})
		return
	}
	out(c, v)
}
func (h *handler) create(c *gin.Context) {
	var q struct{ Username, DisplayName, Password string }
	if c.ShouldBindJSON(&q) != nil {
		bad(c, ErrInvalid)
		return
	}
	pw, e := auth.HashPassword(q.Password)
	if e != nil {
		bad(c, e)
		return
	}
	u := auth.User{Username: strings.ToLower(strings.TrimSpace(q.Username)), DisplayName: q.DisplayName, PasswordHash: pw, Status: "active", Role: "viewer"}
	actor, _ := auth.CurrentUser(c)
	if e = h.s.CreateUser(c, &u, actor.ID); e != nil {
		bad(c, e)
		return
	}
	out(c, UserView{ID: u.ID, Username: u.Username, DisplayName: u.DisplayName, Status: u.Status, Role: u.Role, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt})
}
func (h *handler) update(c *gin.Context) {
	i, ok := id(c)
	if !ok {
		return
	}
	var q struct{ DisplayName, Status string }
	if c.ShouldBindJSON(&q) != nil {
		bad(c, ErrInvalid)
		return
	}
	actor, _ := auth.CurrentUser(c)
	if e := h.s.UpdateUser(c, i, q.DisplayName, q.Status, actor.ID); e != nil {
		bad(c, e)
		return
	}
	out(c, gin.H{"status": "updated"})
}
func (h *handler) status(c *gin.Context) {
	i, ok := id(c)
	if !ok {
		return
	}
	var q struct{ Status string }
	if c.ShouldBindJSON(&q) != nil {
		bad(c, ErrInvalid)
		return
	}
	actor, _ := auth.CurrentUser(c)
	if e := h.s.SetUserStatus(c, i, q.Status, actor.ID); e != nil {
		bad(c, e)
		return
	}
	out(c, gin.H{"status": "updated"})
}
func (h *handler) password(c *gin.Context) {
	i, ok := id(c)
	if !ok {
		return
	}
	var q struct{ Password string }
	if c.ShouldBindJSON(&q) != nil {
		bad(c, ErrInvalid)
		return
	}
	actor, _ := auth.CurrentUser(c)
	if e := h.s.ResetPassword(c, i, q.Password, actor.ID); e != nil {
		bad(c, e)
		return
	}
	out(c, gin.H{"status": "updated"})
}
func (h *handler) authAudit(c *gin.Context) {
	p, z := page(c)
	v, t, e := h.s.AuthAudit(c, p, z, nil, c.Query("event_type"), c.Query("outcome"))
	if e != nil {
		bad(c, e)
		return
	}
	out(c, Page{v, t, p, z})
}
func (h *handler) roleAudit(c *gin.Context) {
	p, z := page(c)
	v, t, e := h.s.RoleAudit(c, p, z)
	if e != nil {
		bad(c, e)
		return
	}
	out(c, Page{v, t, p, z})
}
func (h *handler) sessions(c *gin.Context) {
	p, z := page(c)
	v, t, e := h.s.Sessions(c, p, z, nil)
	if e != nil {
		bad(c, e)
		return
	}
	out(c, Page{v, t, p, z})
}
func (h *handler) userSessions(c *gin.Context) {
	i, ok := id(c)
	if !ok {
		return
	}
	p, z := page(c)
	v, t, e := h.s.Sessions(c, p, z, &i)
	if e != nil {
		bad(c, e)
		return
	}
	out(c, Page{v, t, p, z})
}
func (h *handler) revokeUserSessions(c *gin.Context) {
	i, ok := id(c)
	if !ok {
		return
	}
	a, _ := auth.CurrentUser(c)
	if e := h.s.RevokeSessions(c, i, nil, a.ID); e != nil {
		bad(c, e)
		return
	}
	out(c, gin.H{"status": "revoked"})
}
func (h *handler) config(c *gin.Context) {
	v, e := h.s.Config(c)
	if e != nil {
		bad(c, e)
		return
	}
	out(c, v)
}
func (h *handler) setConfig(c *gin.Context) {
	var v map[string]json.RawMessage
	if c.ShouldBindJSON(&v) != nil {
		bad(c, ErrInvalid)
		return
	}
	a, _ := auth.CurrentUser(c)
	if e := h.s.SetConfig(c, a.ID, v); e != nil {
		bad(c, e)
		return
	}
	out(c, gin.H{"status": "updated"})
}
