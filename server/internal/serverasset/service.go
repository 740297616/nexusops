package serverasset

import (
	"context"
	"time"
)

type Service struct{ repo *Repository }
func NewService(repo *Repository) *Service { return &Service{repo: repo} }
func (s *Service) Create(ctx context.Context, v *Server) error { return s.repo.Create(ctx, v) }
func (s *Service) List(ctx context.Context, p, z int, q, st string) ([]Server, int64, error) { return s.repo.List(ctx, p, z, q, st) }
func (s *Service) Get(ctx context.Context, id string) (Server, error) { return s.repo.Get(ctx, id) }
func (s *Service) Update(ctx context.Context, id, name, description string) error { return s.repo.Update(ctx, id, name, description) }
func (s *Service) SetStatus(ctx context.Context, id, status string) error { return s.repo.SetStatus(ctx, id, status) }
func (s *Service) Delete(ctx context.Context, id string) error { return s.repo.Delete(ctx, id) }
func (s *Service) RegistrationToken(ctx context.Context, id string, actor int64, expiry time.Duration) (string, time.Time, error) { return s.repo.CreateRegistrationToken(ctx, id, actor, expiry) }
func (s *Service) Register(ctx context.Context, token string) (Server, string, error) { return s.repo.Register(ctx, token, time.Now().UTC()) }
func (s *Service) AuthenticateAgent(ctx context.Context, token string) (AgentPrincipal, error) { return s.repo.Credential(ctx, token) }
func (s *Service) Heartbeat(ctx context.Context, id, version string) error { return s.repo.Heartbeat(ctx, id, version, time.Now().UTC()) }
func (s *Service) Report(ctx context.Context, id string, fields map[string]any, version string) error { return s.repo.Report(ctx, id, fields, version, time.Now().UTC()) }
func (s *Service) AgentInfo(ctx context.Context, id string) (AgentCredentialInfo, error) { return s.repo.AgentInfo(ctx, id) }
func (s *Service) RotateCredential(ctx context.Context, id string) (string, error) { return s.repo.Rotate(ctx, id) }
func (s *Service) RevokeCredential(ctx context.Context, id string) error { return s.repo.Revoke(ctx, id) }
func (s *Service) MarkOffline(ctx context.Context, timeout time.Duration) error { return s.repo.MarkOffline(ctx, time.Now().UTC().Add(-timeout)) }

