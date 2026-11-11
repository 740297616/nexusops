package rbac

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/740297616/nexusops/server/internal/auth"
	"github.com/740297616/nexusops/server/internal/platform"
	"github.com/gin-gonic/gin"
)

func RequirePermission(service *Service, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := auth.CurrentUser(c)
		if !ok {
			c.Header("WWW-Authenticate", "Bearer")
			fail(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
			c.Abort()
			return
		}
		allowed, err := service.HasPermission(c.Request.Context(), user.ID, permission)
		if err != nil {
			fail(c, http.StatusServiceUnavailable, "service_unavailable", "Authorization service unavailable")
			c.Abort()
			return
		}
		if !allowed {
			fail(c, http.StatusForbidden, "forbidden", "Permission denied")
			c.Abort()
			return
		}
		c.Next()
	}
}

func RegisterRoutes(router *gin.Engine, authentication *auth.Service, service *Service) {
	h := &handler{service: service}
	group := router.Group("/api/v1")
	group.Use(func(c *gin.Context) { c.Header("Cache-Control", "no-store"); c.Next() }, auth.RequireAuthentication(authentication))
	group.GET("/roles", RequirePermission(service, PermissionRoleRead), h.roles)
	group.GET("/roles/:id", RequirePermission(service, PermissionRoleRead), h.role)
	group.POST("/roles", RequirePermission(service, PermissionRoleCreate), h.createRole)
	group.PUT("/roles/:id", RequirePermission(service, PermissionRoleUpdate), h.updateRole)
	group.PATCH("/roles/:id/status", RequirePermission(service, PermissionRoleUpdate), h.roleStatus)
	group.DELETE("/roles/:id", RequirePermission(service, PermissionRoleDelete), h.deleteRole)
	group.GET("/permissions", RequirePermission(service, PermissionPermissionRead), h.permissions)
	group.GET("/roles/:id/permissions", RequirePermission(service, PermissionPermissionRead), h.rolePermissions)
	group.PUT("/roles/:id/permissions", RequirePermission(service, PermissionPermissionManage), h.setRolePermissions)
	group.GET("/users/:id/roles", RequirePermission(service, PermissionUserRead), h.userRoles)
	group.PUT("/users/:id/roles", RequirePermission(service, PermissionRoleManage), h.replaceRoles)
}

type handler struct{ service *Service }

func fail(c *gin.Context, status int, code, message string) {
	platform.Respond(c, status, nil, &platform.ErrorBody{Code: code, Message: message})
}
func errorResponse(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrUserNotFound):
		fail(c, http.StatusNotFound, "not_found", "User not found")
	case errors.Is(err, ErrInvalidRole), errors.Is(err, ErrInvalidAssignment):
		fail(c, http.StatusBadRequest, "invalid_request", "Invalid role assignment")
	case errors.Is(err, ErrLastSuperAdmin):
		fail(c, http.StatusConflict, "last_super_admin", "Cannot remove the last active super administrator")
	case errors.Is(err, ErrSystemRole):
		fail(c, http.StatusConflict, "system_role", "System role cannot be changed")
	case errors.Is(err, ErrRoleInUse):
		fail(c, http.StatusConflict, "role_in_use", "Role is still assigned to users")
	case errors.Is(err, ErrPermissionUnavailable):
		fail(c, http.StatusBadRequest, "invalid_request", "Permission is unavailable")
	default:
		fail(c, http.StatusServiceUnavailable, "service_unavailable", "Authorization service unavailable")
	}
}
func (h *handler) roles(c *gin.Context) {
	roles, err := h.service.ListRoles(c.Request.Context())
	if err != nil {
		errorResponse(c, err)
		return
	}
	platform.Respond(c, http.StatusOK, roles, nil)
}
func parseRoleID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		fail(c, http.StatusBadRequest, "invalid_request", "Invalid role ID")
		return 0, false
	}
	return id, true
}
func (h *handler) role(c *gin.Context) {
	id, ok := parseRoleID(c)
	if !ok {
		return
	}
	role, err := h.service.GetRole(c.Request.Context(), id)
	if err != nil {
		errorResponse(c, err)
		return
	}
	platform.Respond(c, http.StatusOK, role, nil)
}
func (h *handler) createRole(c *gin.Context) {
	var req struct {
		Code        string `json:"code"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Code == "" || req.Name == "" {
		fail(c, 400, "invalid_request", "Invalid role")
		return
	}
	actor, _ := auth.CurrentUser(c)
	role := Role{Code: req.Code, Name: req.Name, Description: req.Description, Status: "active"}
	if err := h.service.CreateRole(c.Request.Context(), &role, actor.ID); err != nil {
		errorResponse(c, err)
		return
	}
	platform.Respond(c, http.StatusCreated, role, nil)
}
func (h *handler) updateRole(c *gin.Context) {
	id, ok := parseRoleID(c)
	if !ok {
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		fail(c, 400, "invalid_request", "Invalid role")
		return
	}
	actor, _ := auth.CurrentUser(c)
	if err := h.service.UpdateRole(c.Request.Context(), id, req.Name, req.Description, actor.ID); err != nil {
		errorResponse(c, err)
		return
	}
	platform.Respond(c, 200, gin.H{"status": "updated"}, nil)
}
func (h *handler) roleStatus(c *gin.Context) {
	id, ok := parseRoleID(c)
	if !ok {
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || (req.Status != "active" && req.Status != "disabled") {
		fail(c, 400, "invalid_request", "Invalid role status")
		return
	}
	actor, _ := auth.CurrentUser(c)
	if err := h.service.SetRoleStatus(c.Request.Context(), id, req.Status, actor.ID); err != nil {
		errorResponse(c, err)
		return
	}
	platform.Respond(c, 200, gin.H{"status": "updated"}, nil)
}
func (h *handler) deleteRole(c *gin.Context) {
	id, ok := parseRoleID(c)
	if !ok {
		return
	}
	actor, _ := auth.CurrentUser(c)
	if err := h.service.DeleteRole(c.Request.Context(), id, actor.ID); err != nil {
		errorResponse(c, err)
		return
	}
	platform.Respond(c, 200, gin.H{"status": "deleted"}, nil)
}
func (h *handler) permissions(c *gin.Context) {
	permissions, err := h.service.ListPermissions(c.Request.Context())
	if err != nil {
		errorResponse(c, err)
		return
	}
	platform.Respond(c, http.StatusOK, permissions, nil)
}
func (h *handler) rolePermissions(c *gin.Context) {
	id, ok := parseRoleID(c)
	if !ok {
		return
	}
	items, err := h.service.RolePermissions(c.Request.Context(), id)
	if err != nil {
		errorResponse(c, err)
		return
	}
	platform.Respond(c, 200, items, nil)
}
func (h *handler) setRolePermissions(c *gin.Context) {
	id, ok := parseRoleID(c)
	if !ok {
		return
	}
	var req struct {
		PermissionIDs *[]int64 `json:"permission_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PermissionIDs == nil {
		fail(c, 400, "invalid_request", "Invalid permission assignment")
		return
	}
	actor, _ := auth.CurrentUser(c)
	if err := h.service.SetRolePermissions(c.Request.Context(), id, actor.ID, *req.PermissionIDs); err != nil {
		errorResponse(c, err)
		return
	}
	platform.Respond(c, 200, gin.H{"status": "updated"}, nil)
}
func parseUserID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		fail(c, http.StatusBadRequest, "invalid_request", "Invalid user ID")
		return 0, false
	}
	return id, true
}
func (h *handler) userRoles(c *gin.Context) {
	id, ok := parseUserID(c)
	if !ok {
		return
	}
	roles, err := h.service.UserRoles(c.Request.Context(), id)
	if err != nil {
		errorResponse(c, err)
		return
	}
	platform.Respond(c, http.StatusOK, roles, nil)
}
func (h *handler) replaceRoles(c *gin.Context) {
	id, ok := parseUserID(c)
	if !ok {
		return
	}
	var request struct {
		RoleIDs *[]int64 `json:"role_ids"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	if err := c.ShouldBindJSON(&request); err != nil || request.RoleIDs == nil {
		fail(c, http.StatusBadRequest, "invalid_request", "Invalid role assignment")
		return
	}
	actor, _ := auth.CurrentUser(c)
	if err := h.service.ReplaceRoles(c.Request.Context(), actor.ID, id, *request.RoleIDs); err != nil {
		errorResponse(c, err)
		return
	}
	platform.Respond(c, http.StatusOK, gin.H{"status": "updated"}, nil)
}
