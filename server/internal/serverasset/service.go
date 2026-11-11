package serverasset

import (
	"context"
	"time"
)

type Service struct{ repo *Repository }
func NewService(repo *Repository) *Service { return &Service{repo: repo} }
func (s *Service) Create(ctx context.Context, v *Server) error { return s.repo.Create(ctx, v) }
func (s *Service) List(ctx context.Context, p, z int, q, st string, groupID, tagID int64) ([]Server, int64, error) { return s.repo.List(ctx, p, z, q, st, groupID, tagID) }
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


func (s *Service) ListGroups(ctx context.Context, page, size int) ([]Group, int64, error) { return s.repo.ListGroups(ctx, page, size) }
func (s *Service) CreateGroup(ctx context.Context, g *Group) error { return s.repo.CreateGroup(ctx, g) }
func (s *Service) UpdateGroup(ctx context.Context, id int64, name, description string) error { return s.repo.UpdateGroup(ctx, id, name, description) }
func (s *Service) DeleteGroup(ctx context.Context, id int64) error { return s.repo.DeleteGroup(ctx, id) }
func (s *Service) ListTags(ctx context.Context) ([]Tag, error) { return s.repo.ListTags(ctx) }
func (s *Service) CreateTag(ctx context.Context, t *Tag) error { return s.repo.CreateTag(ctx, t) }
func (s *Service) UpdateTag(ctx context.Context, id int64, name string) error { return s.repo.UpdateTag(ctx, id, name) }
func (s *Service) DeleteTag(ctx context.Context, id int64) error { return s.repo.DeleteTag(ctx, id) }
func (s *Service) SetGroups(ctx context.Context, id string, ids []int64) error { return s.repo.SetGroups(ctx, id, ids) }
func (s *Service) SetTags(ctx context.Context, id string, ids []int64) error { return s.repo.SetTags(ctx, id, ids) }
func (s *Service) ServerGroups(ctx context.Context, id string) ([]Group,error) { return s.repo.ServerGroups(ctx,id) }
func (s *Service) ServerTags(ctx context.Context, id string) ([]Tag,error) { return s.repo.ServerTags(ctx,id) }
