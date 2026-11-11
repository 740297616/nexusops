package auth

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
)

type AuditEvent struct {
	UserID    *int64
	EventType string
	Outcome   string
	SourceIP  string
	UserAgent string
	SessionID *int64
	Metadata  map[string]any
}

type RequestMeta struct{ SourceIP, UserAgent string }

func (r *Repository) RecordAudit(ctx context.Context, event AuditEvent) error {
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return err
	}
	var ip any
	if parsed := net.ParseIP(event.SourceIP); parsed != nil {
		ip = event.SourceIP
	}
	return r.db.WithContext(ctx).Exec(`INSERT INTO auth_audit_events (user_id,event_type,outcome,source_ip,user_agent,session_id,metadata) VALUES (?, ?, ?, ?::inet, ?, ?, ?::jsonb)`, event.UserID, event.EventType, event.Outcome, ip, truncateUserAgent(event.UserAgent), event.SessionID, string(metadata)).Error
}

func truncateUserAgent(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 512 {
		return value[:512]
	}
	return value
}

func (s *Service) RecordAudit(ctx context.Context, event AuditEvent) {
	if err := s.repo.RecordAudit(ctx, event); err != nil {
		log.Printf("authentication audit write failed event=%s outcome=%s: %v", event.EventType, event.Outcome, err)
	}
}

func (s *Service) RequestMeta(r *http.Request) RequestMeta {
	remote := r.RemoteAddr
	if host, _, err := net.SplitHostPort(remote); err == nil {
		remote = host
	}
	if parsed := net.ParseIP(remote); parsed != nil && s.isTrustedProxy(parsed) {
		for _, candidate := range strings.Split(r.Header.Get("X-Forwarded-For"), ",") {
			candidate = strings.TrimSpace(candidate)
			if ip := net.ParseIP(candidate); ip != nil && !s.isTrustedProxy(ip) {
				remote = ip.String()
				break
			}
		}
		if remote == r.RemoteAddr || remote == "" {
			if candidate := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP"))); candidate != nil {
				remote = candidate.String()
			}
		}
	}
	return RequestMeta{SourceIP: remote, UserAgent: truncateUserAgent(r.UserAgent())}
}

func (s *Service) isTrustedProxy(ip net.IP) bool {
	for _, network := range s.trustedProxies {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
func (s *Service) auditRequest(ctx context.Context, r *http.Request, eventType, outcome string, userID, sessionID *int64, metadata map[string]any) {
	meta := s.RequestMeta(r)
	s.RecordAudit(ctx, AuditEvent{UserID: userID, EventType: eventType, Outcome: outcome, SourceIP: meta.SourceIP, UserAgent: meta.UserAgent, SessionID: sessionID, Metadata: metadata})
}
