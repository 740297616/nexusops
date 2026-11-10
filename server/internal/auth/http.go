package auth

import (
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/740297616/nexusops/server/internal/platform"
	"github.com/gin-gonic/gin"
)

type loginLimiter struct {
	mu    sync.Mutex
	start time.Time
	count int
}

func (l *loginLimiter) allow(now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.start.IsZero() || now.Sub(l.start) >= time.Minute {
		l.start = now
		l.count = 0
	}
	if l.count >= 20 {
		return false
	}
	l.count++
	return true
}

type principal struct {
	user      User
	sessionID int64
}

func RegisterRoutes(router *gin.Engine, service *Service) {
	h := &handler{service: service}
	routes := router.Group("/api/v1/auth")
	routes.Use(func(c *gin.Context) { c.Header("Cache-Control", "no-store"); c.Header("Pragma", "no-cache"); c.Next() })
	routes.POST("/login", h.login)
	routes.POST("/refresh", h.refresh)
	protected := routes.Group("")
	protected.Use(RequireAuthentication(service))
	protected.POST("/logout", h.logout)
	protected.GET("/me", h.me)
}

type handler struct {
	service *Service
	limiter loginLimiter
}

func respondError(c *gin.Context, status int, code, message string) {
	platform.Respond(c, status, nil, &platform.ErrorBody{Code: code, Message: message})
}
func handleError(c *gin.Context, err error) {
	if errors.Is(err, ErrInvalid) {
		c.Header("WWW-Authenticate", "Bearer")
		respondError(c, http.StatusUnauthorized, "unauthorized", "Invalid or expired credentials")
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
	if !h.limiter.allow(time.Now()) {
		c.Header("Retry-After", "60")
		respondError(c, http.StatusTooManyRequests, "rate_limited", "Too many login attempts")
		return
	}
	var request struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !bindJSON(c, &request) {
		return
	}
	tokens, err := h.service.Login(c.Request.Context(), request.Username, request.Password)
	if err != nil {
		handleError(c, err)
		return
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
		handleError(c, err)
		return
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
	platform.Respond(c, http.StatusOK, gin.H{"status": "logged_out"}, nil)
}
func (h *handler) me(c *gin.Context) {
	identity := c.MustGet("auth.principal").(principal)
	platform.Respond(c, http.StatusOK, identity.user, nil)
}
