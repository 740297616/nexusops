package auth

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/740297616/nexusops/server/internal/platform"
	"github.com/gin-gonic/gin"
)

type principal struct {
	user      User
	sessionID int64
}
type AuthorizationLoader func(context.Context, int64) ([]string, []string, error)

func RegisterRoutes(router *gin.Engine, service *Service, load AuthorizationLoader) {
	h := &handler{service: service, load: load}
	routes := router.Group("/api/v1/auth")
	routes.Use(func(c *gin.Context) { c.Header("Cache-Control", "no-store"); c.Header("Pragma", "no-cache"); c.Next() })
	routes.POST("/login", h.login)
	routes.POST("/refresh", h.refresh)
	protected := routes.Group("")
	protected.Use(RequireAuthentication(service))
	protected.POST("/logout", h.logout)
	protected.GET("/me", h.me)
	protected.GET("/sessions", h.sessions)
	protected.DELETE("/sessions/:id", h.revokeSession)
	protected.DELETE("/sessions", h.revokeOtherSessions)
}

type handler struct {
	service *Service
	load    AuthorizationLoader
}

func respondError(c *gin.Context, status int, code, message string) {
	platform.Respond(c, status, nil, &platform.ErrorBody{Code: code, Message: message})
}
func handleError(c *gin.Context, err error) {
	if errors.Is(err, ErrInvalid) || errors.Is(err, ErrRefreshReplay) {
		c.Header("WWW-Authenticate", "Bearer")
		respondError(c, http.StatusUnauthorized, "unauthorized", "Invalid or expired credentials")
		return
	}
	if errors.Is(err, ErrRateLimited) {
		c.Header("Retry-After", "60")
		respondError(c, http.StatusTooManyRequests, "rate_limited", "Too many login attempts")
		return
	}
	respondError(c, http.StatusServiceUnavailable, "service_unavailable", "Authentication service unavailable")
}
func bindJSON(c *gin.Context, value any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	if err := c.ShouldBindJSON(value); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", "Invalid request")
		return false
	}
	return true
}
func (h *handler) login(c *gin.Context) {
	var request struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !bindJSON(c, &request) {
		return
	}
	meta := h.service.RequestMeta(c.Request)
	allowed, err := h.service.AllowLogin(c.Request.Context(), meta.SourceIP, request.Username)
	if err != nil {
		h.service.auditRequest(c.Request.Context(), c.Request, "login_rate_limit_error", "failure", nil, nil, nil)
		handleError(c, err)
		return
	}
	if !allowed {
		h.service.auditRequest(c.Request.Context(), c.Request, "login_rate_limited", "denied", nil, nil, nil)
		handleError(c, ErrRateLimited)
		return
	}
	tokens, err := h.service.Login(c.Request.Context(), request.Username, request.Password)
	if err != nil {
		h.service.auditRequest(c.Request.Context(), c.Request, "login_failed", "failure", nil, nil, nil)
		handleError(c, err)
		return
	}
	user, sessionID, authErr := h.service.Authenticate(c.Request.Context(), tokens.AccessToken)
	if authErr == nil {
		h.service.auditRequest(c.Request.Context(), c.Request, "login_succeeded", "success", &user.ID, &sessionID, nil)
	}
	platform.Respond(c, http.StatusOK, tokens, nil)
}
func (h *handler) refresh(c *gin.Context) {
	var request struct {
		RefreshToken string `json:"refresh_token"`
	}
	if !bindJSON(c, &request) {
		return
	}
	tokens, err := h.service.Refresh(c.Request.Context(), request.RefreshToken)
	if err != nil {
		event := "refresh_failed"
		if errors.Is(err, ErrRefreshReplay) {
			event = "refresh_replay"
		}
		h.service.auditRequest(c.Request.Context(), c.Request, event, "failure", nil, nil, nil)
		handleError(c, err)
		return
	}
	user, sessionID, authErr := h.service.Authenticate(c.Request.Context(), tokens.AccessToken)
	if authErr == nil {
		h.service.auditRequest(c.Request.Context(), c.Request, "refresh_succeeded", "success", &user.ID, &sessionID, nil)
	}
	platform.Respond(c, http.StatusOK, tokens, nil)
}
func RequireAuthentication(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			handleError(c, ErrInvalid)
			c.Abort()
			return
		}
		user, id, err := service.Authenticate(c.Request.Context(), parts[1])
		if err != nil {
			handleError(c, err)
			c.Abort()
			return
		}
		c.Set("auth.principal", principal{user: user, sessionID: id})
		c.Next()
	}
}
func CurrentUser(c *gin.Context) (User, bool) {
	value, ok := c.Get("auth.principal")
	if !ok {
		return User{}, false
	}
	identity, ok := value.(principal)
	return identity.user, ok
}
func (h *handler) logout(c *gin.Context) {
	identity := c.MustGet("auth.principal").(principal)
	if err := h.service.Logout(c.Request.Context(), identity.sessionID); err != nil {
		handleError(c, err)
		return
	}
	h.service.auditRequest(c.Request.Context(), c.Request, "logout", "success", &identity.user.ID, &identity.sessionID, nil)
	platform.Respond(c, http.StatusOK, gin.H{"status": "logged_out"}, nil)
}
func (h *handler) me(c *gin.Context) {
	identity := c.MustGet("auth.principal").(principal)
	roles, permissions, err := h.load(c.Request.Context(), identity.user.ID)
	if err != nil {
		handleError(c, err)
		return
	}
	platform.Respond(c, http.StatusOK, struct {
		User
		Roles       []string `json:"roles"`
		Permissions []string `json:"permissions"`
	}{identity.user, roles, permissions}, nil)
}
func (h *handler) sessions(c *gin.Context) {
	identity := c.MustGet("auth.principal").(principal)
	sessions, err := h.service.Sessions(c.Request.Context(), identity.user.ID, identity.sessionID)
	if err != nil {
		handleError(c, err)
		return
	}
	platform.Respond(c, http.StatusOK, sessions, nil)
}
func (h *handler) revokeSession(c *gin.Context) {
	identity := c.MustGet("auth.principal").(principal)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_request", "Invalid session ID")
		return
	}
	if id == identity.sessionID {
		respondError(c, http.StatusBadRequest, "invalid_request", "Use logout to revoke the current session")
		return
	}
	if err := h.service.RevokeSession(c.Request.Context(), identity.user.ID, id); err != nil {
		handleError(c, err)
		return
	}
	h.service.auditRequest(c.Request.Context(), c.Request, "session_revoked", "success", &identity.user.ID, &id, nil)
	platform.Respond(c, http.StatusOK, gin.H{"status": "revoked"}, nil)
}
func (h *handler) revokeOtherSessions(c *gin.Context) {
	identity := c.MustGet("auth.principal").(principal)
	if err := h.service.RevokeOtherSessions(c.Request.Context(), identity.user.ID, identity.sessionID); err != nil {
		handleError(c, err)
		return
	}
	h.service.auditRequest(c.Request.Context(), c.Request, "other_sessions_revoked", "success", &identity.user.ID, &identity.sessionID, nil)
	platform.Respond(c, http.StatusOK, gin.H{"status": "revoked"}, nil)
}
