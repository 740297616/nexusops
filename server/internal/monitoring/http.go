package monitoring

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/740297616/nexusops/server/internal/auth"
	"github.com/740297616/nexusops/server/internal/platform"
	"github.com/740297616/nexusops/server/internal/rbac"
	"github.com/740297616/nexusops/server/internal/serverasset"
	"github.com/gin-gonic/gin"
)

type Options struct {
	Audit           *auth.Service
	RateLimitWindow time.Duration
	RateLimit       int
}

func RegisterRoutes(router *gin.Engine, authentication *auth.Service, permissions *rbac.Service, assets *serverasset.Service, service *Service, options Options) {
	h := &handler{assets: assets, service: service, audit: options.Audit, window: options.RateLimitWindow, limit: options.RateLimit}
	agent := router.Group("/api/v1/agent")
	agent.Use(serverasset.RequireAgentAuthentication(assets, options.Audit))
	agent.POST("/metrics", h.ingest)
	admin := router.Group("/api/v1")
	admin.Use(auth.RequireAuthentication(authentication))
	admin.GET("/servers/:id/metrics/latest", rbac.RequirePermission(permissions, rbac.PermissionServerRead), h.latest)
	admin.GET("/servers/:id/metrics/history", rbac.RequirePermission(permissions, rbac.PermissionServerRead), h.history)
}

type handler struct {
	assets  *serverasset.Service
	service *Service
	audit   *auth.Service
	window  time.Duration
	limit   int
}

func fail(c *gin.Context, status int, code, message string) {
	platform.Respond(c, status, nil, &platform.ErrorBody{Code: code, Message: message})
}
func (h *handler) ingest(c *gin.Context) {
	p, ok := serverasset.CurrentAgent(c)
	if !ok {
		fail(c, 401, "unauthorized", "Agent authentication required")
		return
	}
	if h.limit > 0 && h.window > 0 {
		allowed, err := h.assets.AllowAgentRequest(c, "metrics:server:"+p.ServerID, time.Now().UTC(), h.window, h.limit)
		if err != nil {
			if h.audit != nil {
				m := h.audit.RequestMeta(c.Request)
				h.audit.RecordAudit(c, auth.AuditEvent{EventType: "agent.rate_limit.error", Outcome: "failure", SourceIP: m.SourceIP, UserAgent: m.UserAgent, Metadata: map[string]any{"server_id": p.ServerID, "agent_id": p.CredentialID}})
			}
			fail(c, 503, "service_unavailable", "Metric rate limit unavailable")
			return
		}
		if !allowed {
			if h.audit != nil {
				m := h.audit.RequestMeta(c.Request)
				h.audit.RecordAudit(c, auth.AuditEvent{EventType: "agent.rate_limit.exceeded", Outcome: "denied", SourceIP: m.SourceIP, UserAgent: m.UserAgent, Metadata: map[string]any{"server_id": p.ServerID, "agent_id": p.CredentialID}})
			}
			fail(c, 429, "rate_limited", "Metric request rate limited")
			return
		}
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 256*1024)
	var req IngestRequest
	if c.ShouldBindJSON(&req) != nil {
		fail(c, 400, "invalid_request", "Invalid metric request")
		return
	}
	if req.ProtocolVersion != "1" || req.SchemaVersion != 1 || len(req.RequestID) < 8 || len(req.RequestID) > 128 || len(req.Samples) == 0 || len(req.Samples) > 500 {
		fail(c, 400, "invalid_request", "Invalid metric request")
		return
	}
	now := time.Now().UTC()
	if req.SampledAt.Before(now.Add(-5*time.Minute)) || req.SampledAt.After(now.Add(5*time.Minute)) {
		fail(c, 400, "invalid_request", "Metric timestamp is outside the accepted window")
		return
	}
	for _, item := range req.Samples {
		if item.Value < 0 || item.Unit == "" || item.MetricName == "" || item.MetricType == "" || len(item.Labels) > 8 {
			fail(c, 400, "invalid_request", "Invalid metric sample")
			return
		}
	}
	accepted, err := h.service.Ingest(c, p, req)
	if errors.Is(err, ErrDuplicate) {
		platform.Respond(c, 200, IngestResult{Accepted: 0, ReceivedAt: now, RequestID: req.RequestID}, nil)
		return
	}
	if errors.Is(err, ErrInvalid) {
		fail(c, 400, "invalid_request", "Invalid metric sample")
		return
	}
	if err != nil {
		fail(c, 503, "service_unavailable", "Metric storage unavailable")
		return
	}
	if h.audit != nil {
		m := h.audit.RequestMeta(c.Request)
		h.audit.RecordAudit(c, auth.AuditEvent{EventType: "agent.metrics", Outcome: "success", SourceIP: m.SourceIP, UserAgent: m.UserAgent, Metadata: map[string]any{"server_id": p.ServerID, "agent_id": p.CredentialID, "accepted": accepted}})
	}
	platform.Respond(c, 200, IngestResult{Accepted: accepted, ReceivedAt: now, RequestID: req.RequestID}, nil)
}
func parseServerID(c *gin.Context) (string, bool) {
	id := strings.TrimSpace(c.Param("id"))
	if len(id) < 8 || len(id) > 64 {
		fail(c, 400, "invalid_request", "Invalid server ID")
		return "", false
	}
	return id, true
}
func (h *handler) latest(c *gin.Context) {
	id, ok := parseServerID(c)
	if !ok {
		return
	}
	server, err := h.assets.Get(c, id)
	if errors.Is(err, serverasset.ErrNotFound) {
		fail(c, 404, "not_found", "Server not found")
		return
	}
	if err != nil {
		fail(c, 503, "service_unavailable", "Server service unavailable")
		return
	}
	value, err := h.service.Latest(c, id, server.Status)
	if err != nil {
		fail(c, 503, "service_unavailable", "Metric query unavailable")
		return
	}
	platform.Respond(c, 200, value, nil)
}
func (h *handler) history(c *gin.Context) {
	id, ok := parseServerID(c)
	if !ok {
		return
	}
	server, err := h.assets.Get(c, id)
	if errors.Is(err, serverasset.ErrNotFound) {
		fail(c, 404, "not_found", "Server not found")
		return
	}
	if err != nil {
		fail(c, 503, "service_unavailable", "Server service unavailable")
		return
	}
	metricType := c.Query("metric_type")
	metricName := c.Query("metric_name")
	if metricType == "" || metricName == "" {
		fail(c, 400, "invalid_request", "metric_type and metric_name are required")
		return
	}
	from, to := time.Now().UTC().Add(-time.Hour), time.Now().UTC()
	var parseErr error
	if c.Query("from") != "" {
		from, parseErr = time.Parse(time.RFC3339, c.Query("from"))
		if parseErr != nil {
			fail(c, 400, "invalid_request", "Invalid from timestamp")
			return
		}
	}
	if c.Query("to") != "" {
		to, parseErr = time.Parse(time.RFC3339, c.Query("to"))
		if parseErr != nil {
			fail(c, 400, "invalid_request", "Invalid to timestamp")
			return
		}
	}
	step := c.DefaultQuery("step", "raw")
	maxRange := 7 * 24 * time.Hour
	switch step {
	case "raw":
	case "5m":
		maxRange = 30 * 24 * time.Hour
	case "1h":
		maxRange = 180 * 24 * time.Hour
	default:
		fail(c, 400, "invalid_request", "Invalid history step")
		return
	}
	if to.Before(from) || to.Sub(from) > maxRange {
		fail(c, 400, "invalid_request", "Invalid time range")
		return
	}
	limit := 2000
	if raw := c.Query("limit"); raw != "" {
		limit, _ = strconv.Atoi(raw)
		if limit < 1 || limit > 2000 {
			fail(c, 400, "invalid_request", "Invalid limit")
			return
		}
	}
	labelKey, labelValue := "", ""
	if metricType == "filesystem" {
		labelKey, labelValue = "mountpoint", c.Query("mountpoint")
	}
	if metricType == "network" {
		labelKey, labelValue = "interface", c.Query("interface")
	}
	if labelValue == "" && (metricType == "filesystem" || metricType == "network") {
		fail(c, 400, "invalid_request", "A resource label is required")
		return
	}
	value, err := h.service.History(c, id, server.Status, metricType, metricName, labelKey, labelValue, step, from, to, limit)
	if errors.Is(err, ErrInvalid) {
		fail(c, 400, "invalid_request", "Invalid metric query")
		return
	}
	if err != nil {
		fail(c, 503, "service_unavailable", "Metric query unavailable")
		return
	}
	platform.Respond(c, 200, value, nil)
}
