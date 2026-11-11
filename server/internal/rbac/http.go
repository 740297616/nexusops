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
	group.GET("/permissions", RequirePermission(service, PermissionPermissionRead), h.permissions)
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
func (h *handler) permissions(c *gin.Context) {
	permissions, err := h.service.ListPermissions(c.Request.Context())
	if err != nil {
		errorResponse(c, err)
		return
	}
	platform.Respond(c, http.StatusOK, permissions, nil)
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
