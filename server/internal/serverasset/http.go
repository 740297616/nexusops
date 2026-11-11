package serverasset

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/740297616/nexusops/server/internal/auth"
	"github.com/740297616/nexusops/server/internal/platform"
	"github.com/740297616/nexusops/server/internal/rbac"
	"github.com/gin-gonic/gin"
)

type handler struct {
	service  *Service
	tokenTTL time.Duration
	audit    *auth.Service
}

type AgentOptions struct {
	Audit              *auth.Service
	RateLimitWindow    time.Duration
	RegisterRateLimit  int
	HeartbeatRateLimit int
	ReportRateLimit    int
}

func RegisterRoutes(router *gin.Engine, authentication *auth.Service, permissions *rbac.Service, service *Service, tokenTTL time.Duration) {
	h := &handler{service: service, tokenTTL: tokenTTL, audit: authentication}
	g := router.Group("/api/v1")
	g.Use(auth.RequireAuthentication(authentication))
	g.GET("/servers", rbac.RequirePermission(permissions, rbac.PermissionServerRead), h.list)
	g.GET("/servers/:id", rbac.RequirePermission(permissions, rbac.PermissionServerRead), h.get)
	g.POST("/servers", rbac.RequirePermission(permissions, rbac.PermissionServerCreate), h.create)
	g.PUT("/servers/:id", rbac.RequirePermission(permissions, rbac.PermissionServerUpdate), h.update)
	g.PATCH("/servers/:id/status", rbac.RequirePermission(permissions, rbac.PermissionServerUpdate), h.status)
	g.DELETE("/servers/:id", rbac.RequirePermission(permissions, rbac.PermissionServerDelete), h.delete)
	g.GET("/servers/:id/agent", rbac.RequirePermission(permissions, rbac.PermissionServerRead), h.agent)
	g.POST("/servers/:id/agent/register-token", rbac.RequirePermission(permissions, rbac.PermissionServerAgent), h.registerToken)
	g.POST("/servers/:id/agent/register", rbac.RequirePermission(permissions, rbac.PermissionServerAgent), h.registerToken)
	g.POST("/servers/:id/agent/rotate-token", rbac.RequirePermission(permissions, rbac.PermissionServerAgent), h.rotate)
	g.POST("/servers/:id/agent/revoke", rbac.RequirePermission(permissions, rbac.PermissionServerAgent), h.revoke)
	g.GET("/groups", rbac.RequirePermission(permissions, rbac.PermissionServerRead), h.groups)
	g.POST("/groups", rbac.RequirePermission(permissions, rbac.PermissionServerGroup), h.createGroup)
	g.PUT("/groups/:group_id", rbac.RequirePermission(permissions, rbac.PermissionServerGroup), h.updateGroup)
	g.DELETE("/groups/:group_id", rbac.RequirePermission(permissions, rbac.PermissionServerGroup), h.deleteGroup)
	g.GET("/tags", rbac.RequirePermission(permissions, rbac.PermissionServerRead), h.tags)
	g.POST("/tags", rbac.RequirePermission(permissions, rbac.PermissionServerTag), h.createTag)
	g.PUT("/tags/:tag_id", rbac.RequirePermission(permissions, rbac.PermissionServerTag), h.updateTag)
	g.DELETE("/tags/:tag_id", rbac.RequirePermission(permissions, rbac.PermissionServerTag), h.deleteTag)
	g.GET("/servers/:id/groups", rbac.RequirePermission(permissions, rbac.PermissionServerRead), h.serverGroups)
	g.PUT("/servers/:id/groups", rbac.RequirePermission(permissions, rbac.PermissionServerGroup), h.setGroups)
	g.GET("/servers/:id/tags", rbac.RequirePermission(permissions, rbac.PermissionServerRead), h.serverTags)
	g.PUT("/servers/:id/tags", rbac.RequirePermission(permissions, rbac.PermissionServerTag), h.setTags)
}

func RegisterAgentRoutes(router *gin.Engine, service *Service, options AgentOptions) {
	h := &agentHandler{service: service, audit: options.Audit, window: options.RateLimitWindow, registerLimit: options.RegisterRateLimit, heartbeatLimit: options.HeartbeatRateLimit, reportLimit: options.ReportRateLimit}
	g := router.Group("/api/v1/agent")
	g.POST("/register", h.register)
	g.Use(agentAuth(service, options.Audit))
	g.POST("/heartbeat", h.heartbeat)
	g.POST("/report", h.report)
}

func fail(c *gin.Context, status int, code, message string) {
	platform.Respond(c, status, nil, &platform.ErrorBody{Code: code, Message: message})
}
func parseID(c *gin.Context) (string, bool) {
	id := strings.TrimSpace(c.Param("id"))
	if len(id) < 8 || len(id) > 64 {
		fail(c, 400, "invalid_request", "Invalid server ID")
		return "", false
	}
	return id, true
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
func (h *handler) list(c *gin.Context) {
	p, z := page(c)
	groupID, _ := strconv.ParseInt(c.Query("group_id"), 10, 64)
	tagID, _ := strconv.ParseInt(c.Query("tag_id"), 10, 64)
	items, total, err := h.service.List(c, p, z, c.Query("q"), c.Query("status"), groupID, tagID)
	if err != nil {
		fail(c, 500, "service_unavailable", "Server service unavailable")
		return
	}
	platform.Respond(c, 200, Page{items, total, p, z}, nil)
}
func (h *handler) get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	s, err := h.service.Get(c, id)
	if errors.Is(err, ErrNotFound) {
		fail(c, 404, "not_found", "Server not found")
		return
	}
	if err != nil {
		fail(c, 500, "service_unavailable", "Server service unavailable")
		return
	}
	platform.Respond(c, 200, s, nil)
}
func (h *handler) record(c *gin.Context, event string, actor int64, metadata map[string]any) {
	h.audit.RecordAudit(c, auth.AuditEvent{UserID: &actor, EventType: event, Outcome: "success", SourceIP: h.audit.RequestMeta(c.Request).SourceIP, UserAgent: h.audit.RequestMeta(c.Request).UserAgent, Metadata: metadata})
}
func (h *handler) create(c *gin.Context) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if c.ShouldBindJSON(&req) != nil || strings.TrimSpace(req.Name) == "" {
		fail(c, 400, "invalid_request", "Invalid server")
		return
	}
	s := Server{Name: req.Name, Description: req.Description}
	if err := h.service.Create(c, &s); err != nil {
		fail(c, 400, "invalid_request", "Invalid server")
		return
	}
	if u, ok := auth.CurrentUser(c); ok {
		h.record(c, "server_created", u.ID, map[string]any{"server_id": s.ID})
	}
	platform.Respond(c, 201, s, nil)
}
func (h *handler) update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if c.ShouldBindJSON(&req) != nil {
		fail(c, 400, "invalid_request", "Invalid server")
		return
	}
	if err := h.service.Update(c, id, req.Name, req.Description); err != nil {
		if errors.Is(err, ErrNotFound) {
			fail(c, 404, "not_found", "Server not found")
		} else {
			fail(c, 400, "invalid_request", "Invalid server")
		}
		return
	}
	if u, ok := auth.CurrentUser(c); ok {
		h.record(c, "server_updated", u.ID, map[string]any{"server_id": id})
	}
	platform.Respond(c, 200, gin.H{"status": "updated"}, nil)
}
func (h *handler) status(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if c.ShouldBindJSON(&req) != nil {
		fail(c, 400, "invalid_request", "Invalid status")
		return
	}
	if err := h.service.SetStatus(c, id, req.Status); err != nil {
		if errors.Is(err, ErrNotFound) {
			fail(c, 404, "not_found", "Server not found")
		} else {
			fail(c, 400, "invalid_request", "Invalid status")
		}
		return
	}
	if u, ok := auth.CurrentUser(c); ok {
		h.record(c, "server_status_changed", u.ID, map[string]any{"server_id": id, "status": req.Status})
	}
	platform.Respond(c, 200, gin.H{"status": "updated"}, nil)
}
func (h *handler) delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			fail(c, 404, "not_found", "Server not found")
		} else if errors.Is(err, ErrAgentAttached) {
			fail(c, 409, "agent_attached", "Revoke the Agent credential before deleting the server")
		} else {
			fail(c, 500, "service_unavailable", "Server service unavailable")
		}
		return
	}
	if u, ok := auth.CurrentUser(c); ok {
		h.record(c, "server_deleted", u.ID, map[string]any{"server_id": id})
	}
	platform.Respond(c, 200, gin.H{"status": "deleted"}, nil)
}
func (h *handler) agent(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	v, err := h.service.AgentInfo(c, id)
	if err != nil {
		fail(c, 500, "service_unavailable", "Agent service unavailable")
		return
	}
	platform.Respond(c, 200, v, nil)
}
func (h *handler) registerToken(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	u, _ := auth.CurrentUser(c)
	token, expires, err := h.service.RegistrationToken(c, id, u.ID, h.tokenTTL)
	if errors.Is(err, ErrNotFound) {
		fail(c, 404, "not_found", "Server not found")
		return
	}
	if err != nil {
		fail(c, 409, "agent_registration_unavailable", "Unable to create registration token")
		return
	}
	h.record(c, "agent_registration_token_created", u.ID, map[string]any{"server_id": id, "expires_at": expires})
	platform.Respond(c, 201, gin.H{"token": token, "expires_at": expires}, nil)
}
func (h *handler) rotate(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	token, err := h.service.RotateCredential(c, id)
	if err != nil {
		fail(c, 409, "agent_unavailable", "No active Agent credential")
		return
	}
	if u, ok := auth.CurrentUser(c); ok {
		h.record(c, "agent.credential.rotate", u.ID, map[string]any{"server_id": id})
	}
	platform.Respond(c, 200, gin.H{"credential": token}, nil)
}
func (h *handler) revoke(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.service.RevokeCredential(c, id); err != nil {
		fail(c, 500, "service_unavailable", "Agent service unavailable")
		return
	}
	if u, ok := auth.CurrentUser(c); ok {
		h.record(c, "agent.credential.revoke", u.ID, map[string]any{"server_id": id})
	}
	platform.Respond(c, 200, gin.H{"status": "revoked"}, nil)
}

type agentHandler struct {
	service                                    *Service
	audit                                      *auth.Service
	window                                     time.Duration
	registerLimit, heartbeatLimit, reportLimit int
}

func (h *agentHandler) auditEvent(c *gin.Context, event, outcome string, p *AgentPrincipal, metadata map[string]any) {
	if h.audit == nil {
		return
	}
	if p != nil {
		if metadata == nil {
			metadata = map[string]any{}
		}
		metadata["server_id"] = p.ServerID
		metadata["agent_id"] = p.CredentialID
	}
	meta := h.audit.RequestMeta(c.Request)
	h.audit.RecordAudit(c, auth.AuditEvent{EventType: event, Outcome: outcome, SourceIP: meta.SourceIP, UserAgent: meta.UserAgent, Metadata: metadata})
}
func (h *agentHandler) allow(c *gin.Context, key string, limit int, p *AgentPrincipal) bool {
	if limit < 1 || h.window <= 0 {
		return true
	}
	allowed, err := h.service.repo.AllowAgentRequest(c, key, time.Now().UTC(), h.window, limit)
	if err != nil {
		h.auditEvent(c, "agent.rate_limit.error", "failure", p, nil)
		fail(c, 503, "service_unavailable", "Agent rate limit unavailable")
		return false
	}
	if !allowed {
		h.auditEvent(c, "agent.rate_limit.exceeded", "denied", p, nil)
		c.Header("Retry-After", strconv.Itoa(int(h.window.Seconds())))
		fail(c, 429, "rate_limited", "Agent request rate limited")
		return false
	}
	return true
}
func agentAuth(service *Service, audit *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Agent") {
			if audit != nil {
				m := audit.RequestMeta(c.Request)
				audit.RecordAudit(c, auth.AuditEvent{EventType: "agent.authentication.failed", Outcome: "failure", SourceIP: m.SourceIP, UserAgent: m.UserAgent})
			}
			fail(c, 401, "unauthorized", "Agent authentication required")
			c.Abort()
			return
		}
		p, err := service.AuthenticateAgent(c, parts[1])
		if err != nil {
			if audit != nil {
				m := audit.RequestMeta(c.Request)
				audit.RecordAudit(c, auth.AuditEvent{EventType: "agent.authentication.failed", Outcome: "failure", SourceIP: m.SourceIP, UserAgent: m.UserAgent})
			}
			fail(c, 401, "unauthorized", "Invalid Agent credentials")
			c.Abort()
			return
		}
		c.Set("agent.principal", p)
		c.Next()
	}
}
func agentPrincipal(c *gin.Context) (AgentPrincipal, bool) {
	v, ok := c.Get("agent.principal")
	p, good := v.(AgentPrincipal)
	return p, ok && good
}
func (h *agentHandler) register(c *gin.Context) {
	source := "unknown"
	if h.audit != nil {
		source = h.audit.RequestMeta(c.Request).SourceIP
	}
	if !h.allow(c, "register:ip:"+source, h.registerLimit, nil) {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64*1024)
	var req struct {
		ProtocolVersion   string `json:"protocol_version"`
		RegistrationToken string `json:"registration_token"`
		Hostname          string `json:"hostname"`
		OS                string `json:"os"`
		OSVersion         string `json:"os_version"`
		Architecture      string `json:"architecture"`
		KernelVersion     string `json:"kernel_version"`
		CPUModel          string `json:"cpu_model"`
		CPUCores          int    `json:"cpu_cores"`
		MemoryTotal       int64  `json:"memory_total"`
		DiskTotal         int64  `json:"disk_total"`
		PrimaryIP         string `json:"primary_ip"`
		AgentVersion      string `json:"agent_version"`
	}
	if c.ShouldBindJSON(&req) != nil || req.ProtocolVersion != "1" || req.RegistrationToken == "" {
		h.auditEvent(c, "agent.register.failed", "failure", nil, nil)
		fail(c, 400, "invalid_request", "Invalid registration request")
		return
	}
	s, credential, err := h.service.Register(c, req.RegistrationToken)
	if err != nil {
		h.auditEvent(c, "agent.register.failed", "failure", nil, nil)
		fail(c, 401, "registration_failed", "Agent registration failed")
		return
	}
	_ = h.service.Report(c, s.ID, map[string]any{"hostname": req.Hostname, "os": req.OS, "os_version": req.OSVersion, "architecture": req.Architecture, "kernel_version": req.KernelVersion, "cpu_model": req.CPUModel, "cpu_cores": req.CPUCores, "memory_total": req.MemoryTotal, "disk_total": req.DiskTotal, "primary_ip": req.PrimaryIP}, req.AgentVersion)
	info, _ := h.service.AgentInfo(c, s.ID)
	h.auditEvent(c, "agent.register.success", "success", &AgentPrincipal{ServerID: s.ID, CredentialID: info.ID}, nil)
	platform.Respond(c, 200, gin.H{"server_id": s.ID, "credential": credential, "protocol_version": "1"}, nil)
}
func (h *agentHandler) heartbeat(c *gin.Context) {
	p, ok := agentPrincipal(c)
	if !ok {
		fail(c, 401, "unauthorized", "Agent authentication required")
		return
	}
	if !h.allow(c, "heartbeat:server:"+p.ServerID, h.heartbeatLimit, &p) {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16*1024)
	var req struct {
		ProtocolVersion string    `json:"protocol_version"`
		RequestID       string    `json:"request_id"`
		AgentVersion    string    `json:"agent_version"`
		Timestamp       time.Time `json:"timestamp"`
	}
	if c.ShouldBindJSON(&req) != nil || req.ProtocolVersion != "1" {
		h.auditEvent(c, "agent.heartbeat", "failure", &p, nil)
		fail(c, 400, "invalid_request", "Invalid heartbeat")
		return
	}
	if err := h.service.Heartbeat(c, p.ServerID, req.AgentVersion); err != nil {
		h.auditEvent(c, "agent.heartbeat", "failure", &p, nil)
		fail(c, 409, "agent_rejected", "Agent heartbeat rejected")
		return
	}
	h.auditEvent(c, "agent.heartbeat", "success", &p, nil)
	platform.Respond(c, 200, gin.H{"status": "ok", "request_id": req.RequestID}, nil)
}
func (h *agentHandler) report(c *gin.Context) {
	p, ok := agentPrincipal(c)
	if !ok {
		fail(c, 401, "unauthorized", "Agent authentication required")
		return
	}
	if !h.allow(c, "report:server:"+p.ServerID, h.reportLimit, &p) {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64*1024)
	var req struct {
		ProtocolVersion string    `json:"protocol_version"`
		RequestID       string    `json:"request_id"`
		Timestamp       time.Time `json:"timestamp"`
		AgentVersion    string    `json:"agent_version"`
		Hostname        string    `json:"hostname"`
		OS              string    `json:"os"`
		OSVersion       string    `json:"os_version"`
		Architecture    string    `json:"architecture"`
		KernelVersion   string    `json:"kernel_version"`
		CPUModel        string    `json:"cpu_model"`
		CPUCores        int       `json:"cpu_cores"`
		MemoryTotal     int64     `json:"memory_total"`
		DiskTotal       int64     `json:"disk_total"`
		PrimaryIP       string    `json:"primary_ip"`
	}
	if c.ShouldBindJSON(&req) != nil || req.ProtocolVersion != "1" {
		h.auditEvent(c, "agent.report", "failure", &p, nil)
		fail(c, 400, "invalid_request", "Invalid report")
		return
	}
	err := h.service.Report(c, p.ServerID, map[string]any{"hostname": req.Hostname, "os": req.OS, "os_version": req.OSVersion, "architecture": req.Architecture, "kernel_version": req.KernelVersion, "cpu_model": req.CPUModel, "cpu_cores": req.CPUCores, "memory_total": req.MemoryTotal, "disk_total": req.DiskTotal, "primary_ip": req.PrimaryIP}, req.AgentVersion)
	if err != nil {
		h.auditEvent(c, "agent.report", "failure", &p, nil)
		fail(c, 409, "agent_rejected", "Agent report rejected")
		return
	}
	h.auditEvent(c, "agent.report", "success", &p, nil)
	platform.Respond(c, 200, gin.H{"status": "ok", "request_id": req.RequestID}, nil)
}

func idParam(c *gin.Context, key string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(key), 10, 64)
	if err != nil || id <= 0 {
		fail(c, 400, "invalid_request", "Invalid ID")
		return 0, false
	}
	return id, true
}
func (h *handler) groups(c *gin.Context) {
	p, z := page(c)
	items, total, err := h.service.ListGroups(c, p, z)
	if err != nil {
		fail(c, 500, "service_unavailable", "Group service unavailable")
		return
	}
	platform.Respond(c, 200, gin.H{"items": items, "total": total, "page": p, "page_size": z}, nil)
}
func (h *handler) createGroup(c *gin.Context) {
	var req struct {
		Code        string `json:"code"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if c.ShouldBindJSON(&req) != nil {
		fail(c, 400, "invalid_request", "Invalid group")
		return
	}
	g := Group{Code: req.Code, Name: req.Name, Description: req.Description}
	if err := h.service.CreateGroup(c, &g); err != nil {
		fail(c, 400, "invalid_request", "Invalid group")
		return
	}
	platform.Respond(c, 201, g, nil)
}
func (h *handler) updateGroup(c *gin.Context) {
	id, ok := idParam(c, "group_id")
	if !ok {
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if c.ShouldBindJSON(&req) != nil {
		fail(c, 400, "invalid_request", "Invalid group")
		return
	}
	if err := h.service.UpdateGroup(c, id, req.Name, req.Description); err != nil {
		if errors.Is(err, ErrNotFound) {
			fail(c, 404, "not_found", "Group not found")
		} else {
			fail(c, 400, "invalid_request", "Invalid group")
		}
		return
	}
	platform.Respond(c, 200, gin.H{"status": "updated"}, nil)
}
func (h *handler) deleteGroup(c *gin.Context) {
	id, ok := idParam(c, "group_id")
	if !ok {
		return
	}
	if err := h.service.DeleteGroup(c, id); err != nil {
		if errors.Is(err, ErrAssociationInUse) {
			fail(c, 409, "association_in_use", "Group is still assigned to servers")
		} else {
			fail(c, 404, "not_found", "Group not found")
		}
		return
	}
	platform.Respond(c, 200, gin.H{"status": "deleted"}, nil)
}
func (h *handler) tags(c *gin.Context) {
	items, err := h.service.ListTags(c)
	if err != nil {
		fail(c, 500, "service_unavailable", "Tag service unavailable")
		return
	}
	platform.Respond(c, 200, items, nil)
}
func (h *handler) createTag(c *gin.Context) {
	var req struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if c.ShouldBindJSON(&req) != nil {
		fail(c, 400, "invalid_request", "Invalid tag")
		return
	}
	t := Tag{Code: req.Code, Name: req.Name}
	if err := h.service.CreateTag(c, &t); err != nil {
		fail(c, 400, "invalid_request", "Invalid tag")
		return
	}
	platform.Respond(c, 201, t, nil)
}
func (h *handler) updateTag(c *gin.Context) {
	id, ok := idParam(c, "tag_id")
	if !ok {
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if c.ShouldBindJSON(&req) != nil {
		fail(c, 400, "invalid_request", "Invalid tag")
		return
	}
	if err := h.service.UpdateTag(c, id, req.Name); err != nil {
		if errors.Is(err, ErrNotFound) {
			fail(c, 404, "not_found", "Tag not found")
		} else {
			fail(c, 400, "invalid_request", "Invalid tag")
		}
		return
	}
	platform.Respond(c, 200, gin.H{"status": "updated"}, nil)
}
func (h *handler) deleteTag(c *gin.Context) {
	id, ok := idParam(c, "tag_id")
	if !ok {
		return
	}
	if err := h.service.DeleteTag(c, id); err != nil {
		if errors.Is(err, ErrAssociationInUse) {
			fail(c, 409, "association_in_use", "Tag is still assigned to servers")
		} else {
			fail(c, 404, "not_found", "Tag not found")
		}
		return
	}
	platform.Respond(c, 200, gin.H{"status": "deleted"}, nil)
}
func (h *handler) serverGroups(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	items, err := h.service.ServerGroups(c, id)
	if err != nil {
		fail(c, 500, "service_unavailable", "Group service unavailable")
		return
	}
	platform.Respond(c, 200, items, nil)
}
func (h *handler) setGroups(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		IDs []int64 `json:"group_ids"`
	}
	if c.ShouldBindJSON(&req) != nil || len(req.IDs) > 100 {
		fail(c, 400, "invalid_request", "Invalid groups")
		return
	}
	if err := h.service.SetGroups(c, id, req.IDs); err != nil {
		fail(c, 400, "invalid_request", "Invalid groups")
		return
	}
	platform.Respond(c, 200, gin.H{"status": "updated"}, nil)
}
func (h *handler) serverTags(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	items, err := h.service.ServerTags(c, id)
	if err != nil {
		fail(c, 500, "service_unavailable", "Tag service unavailable")
		return
	}
	platform.Respond(c, 200, items, nil)
}
func (h *handler) setTags(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		IDs []int64 `json:"tag_ids"`
	}
	if c.ShouldBindJSON(&req) != nil || len(req.IDs) > 100 {
		fail(c, 400, "invalid_request", "Invalid tags")
		return
	}
	if err := h.service.SetTags(c, id, req.IDs); err != nil {
		fail(c, 400, "invalid_request", "Invalid tags")
		return
	}
	platform.Respond(c, 200, gin.H{"status": "updated"}, nil)
}
