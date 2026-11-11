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
func (s *Service) ListPermissions(ctx context.Context) ([]Permission, error) {
	return s.repo.ListPermissions(ctx)
}
func (s *Service) UserRoles(ctx context.Context, userID int64) ([]Role, error) {
	return s.repo.UserRoles(ctx, userID)
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
