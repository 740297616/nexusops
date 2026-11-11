package monitoring

import (
	"context"
	"log"
	"time"

	"github.com/740297616/nexusops/server/internal/serverasset"
)

type Service struct {
	repo         *Repository
	staleAfter   time.Duration
	rawRetention time.Duration
}

func NewService(repo *Repository, staleAfter, rawRetention time.Duration) *Service {
	return &Service{repo: repo, staleAfter: staleAfter, rawRetention: rawRetention}
}
func (s *Service) Ingest(ctx context.Context, p serverasset.AgentPrincipal, req IngestRequest) (int, error) {
	return s.repo.Insert(ctx, p, req, time.Now().UTC())
}
func (s *Service) Latest(ctx context.Context, id string, status string) (Latest, error) {
	items, err := s.repo.Latest(ctx, id, time.Now().UTC(), s.staleAfter)
	if err != nil {
		return Latest{}, err
	}
	var observed time.Time
	for _, item := range items {
		if item.ReceivedAt.After(observed) {
			observed = item.ReceivedAt
		}
	}
	return Latest{ServerID: id, Status: status, ObservedAt: observed, Stale: observed.IsZero() || time.Since(observed) > s.staleAfter || status != "online", Metrics: items}, nil
}
func (s *Service) History(ctx context.Context, id, status, metricType, metricName, labelKey, labelValue string, from, to time.Time, limit int) (History, error) {
	if metricType != "cpu" && metricType != "memory" && metricType != "filesystem" && metricType != "network" {
		return History{}, ErrInvalid
	}
	if metricType == "filesystem" && labelKey != "mountpoint" {
		return History{}, ErrInvalid
	}
	if metricType == "network" && labelKey != "interface" {
		return History{}, ErrInvalid
	}
	items, err := s.repo.History(ctx, id, metricType, metricName, labelKey, labelValue, from, to, limit)
	if err != nil {
		return History{}, err
	}
	return History{ServerID: id, MetricType: metricType, MetricName: metricName, Aggregation: "raw", Stale: status != "online" || (len(items) > 0 && time.Since(items[len(items)-1].ReceivedAt) > s.staleAfter), Series: items}, nil
}
func (s *Service) Cleanup(ctx context.Context) (int64, error) {
	return s.repo.Cleanup(ctx, time.Now().UTC().Add(-s.rawRetention), 500)
}
func (s *Service) StartCleanup(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if removed, err := s.Cleanup(ctx); err != nil {
					log.Printf("metrics cleanup failed: %v", err)
				} else if removed > 0 {
					log.Printf("metrics cleanup removed records=%d", removed)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
