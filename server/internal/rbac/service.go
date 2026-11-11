package rbac

import (
	"context"
	"errors"
	"sort"
)

var ErrInvalidAssignment = errors.New("invalid role assignment")

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }
func (s *Service) HasPermission(ctx context.Context, userID int64, permission string) (bool, error) {
	return s.repo.HasPermission(ctx, userID, permission)
}
func (s *Service) Effective(ctx context.Context, userID int64) ([]string, []string, error) {
	return s.repo.Effective(ctx, userID)
}
func (s *Service) ListRoles(ctx context.Context) ([]Role, error) { return s.repo.ListRoles(ctx) }
func (s *Service) GetRole(ctx context.Context, id int64) (Role, error) {
	return s.repo.GetRole(ctx, id)
}
func (s *Service) CreateRole(ctx context.Context, role *Role, actorID int64) error {
	return s.repo.CreateRole(ctx, role, actorID)
}
func (s *Service) UpdateRole(ctx context.Context, id int64, name, description string, actorID int64) error {
	return s.repo.UpdateRole(ctx, id, name, description, actorID)
}
func (s *Service) SetRoleStatus(ctx context.Context, id int64, status string, actorID int64) error {
	return s.repo.SetRoleStatusManaged(ctx, id, status, actorID)
}
func (s *Service) DeleteRole(ctx context.Context, id, actorID int64) error {
	return s.repo.DeleteRoleManaged(ctx, id, actorID)
}
func (s *Service) ListPermissions(ctx context.Context) ([]Permission, error) {
	return s.repo.ListPermissions(ctx)
}
func (s *Service) UserRoles(ctx context.Context, userID int64) ([]Role, error) {
	return s.repo.UserRoles(ctx, userID)
}
func (s *Service) RolePermissions(ctx context.Context, roleID int64) ([]Permission, error) {
	return s.repo.RolePermissions(ctx, roleID)
}
func (s *Service) SetRolePermissions(ctx context.Context, roleID, actorID int64, ids []int64) error {
	if len(ids) > 200 {
		return ErrInvalidAssignment
	}
	seen := map[int64]bool{}
	for _, id := range ids {
		if id <= 0 || seen[id] {
			return ErrInvalidAssignment
		}
		seen[id] = true
	}
	return s.repo.SetRolePermissions(ctx, roleID, actorID, ids)
}
func (s *Service) ReplaceRoles(ctx context.Context, actorID, userID int64, ids []int64) error {
	if len(ids) > 100 {
		return ErrInvalidAssignment
	}
	seen := make(map[int64]bool, len(ids))
	normalized := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 || seen[id] {
			return ErrInvalidAssignment
		}
		seen[id] = true
		normalized = append(normalized, id)
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i] < normalized[j] })
	return s.repo.ReplaceRoles(ctx, actorID, userID, normalized)
}
