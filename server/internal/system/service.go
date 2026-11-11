package system

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/740297616/nexusops/server/internal/auth"
)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }
func (s *Service) ListUsers(c context.Context, p, sz int, k, st string) ([]UserView, int64, error) {
	return s.repo.ListUsers(c, p, sz, k, st)
}
func (s *Service) GetUser(c context.Context, id int64) (UserView, error) {
	return s.repo.GetUser(c, id)
}
func (s *Service) CreateUser(c context.Context, u *auth.User, a int64) error {
	return s.repo.CreateUser(c, u, a)
}
func (s *Service) UpdateUser(c context.Context, id int64, d, st string, a int64) error {
	return s.repo.UpdateUser(c, id, d, st, a)
}
func (s *Service) SetUserStatus(c context.Context, id int64, st string, a int64) error {
	return s.repo.SetUserStatus(c, id, st, a)
}
func (s *Service) ResetPassword(c context.Context, id int64, pw string, a int64) error {
	return s.repo.CreatePassword(c, id, pw, a)
}
func (s *Service) Sessions(c context.Context, p, sz int, u *int64) ([]auth.SessionInfo, int64, error) {
	return s.repo.ListSessions(c, p, sz, u)
}
func (s *Service) RevokeSessions(c context.Context, u int64, sid *int64, a int64) error {
	return s.repo.RevokeUserSessions(c, u, sid, a)
}
func (s *Service) AuthAudit(c context.Context, p, sz int, u *int64, e, o string) ([]AuditView, int64, error) {
	return s.repo.ListAuthAudit(c, p, sz, u, e, o)
}
func (s *Service) RoleAudit(c context.Context, p, sz int) ([]RoleAuditView, int64, error) {
	return s.repo.ListRoleAudit(c, p, sz)
}
func (s *Service) Config(c context.Context) ([]ConfigView, error) { return s.repo.Config(c) }
func (s *Service) SetConfig(c context.Context, a int64, v map[string]json.RawMessage) error {
	return s.repo.SetConfig(c, a, v)
}

var ErrInvalid = errors.New("invalid management request")
