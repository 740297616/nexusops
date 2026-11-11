package monitoring

import (
	"context"
	"log"
	"time"

	"github.com/740297616/nexusops/server/internal/serverasset"
)

type Service struct {
	repo                                 *Repository
	staleAfter                           time.Duration
	rawRetention                         time.Duration
	rollup5mRetention, rollup1hRetention time.Duration
}

func NewService(repo *Repository, staleAfter, rawRetention, rollup5mRetention, rollup1hRetention time.Duration) *Service {
	return &Service{repo: repo, staleAfter: staleAfter, rawRetention: rawRetention, rollup5mRetention: rollup5mRetention, rollup1hRetention: rollup1hRetention}
}
func (s *Service) Ingest(ctx context.Context, p serverasset.AgentPrincipal, req IngestRequest) (int, error) {
	return s.repo.Insert(ctx, p, req, time.Now().UTC())
}
func (s *Service) Latest(ctx context.Context, id string, status string) (Latest, error) {
	items, err := s.repo.Latest(ctx, id, time.Now().UTC(), s.staleAfter)
	if err != nil {
		return Latest{}, err
	}
	var sampled, received time.Time
	for _, item := range items {
		if item.SampledAt.After(sampled) {
			sampled = item.SampledAt
		}
		if item.ReceivedAt.After(received) {
			received = item.ReceivedAt
		}
	}
	metricsStatus := metricFreshness(sampled, time.Now().UTC(), s.staleAfter)
	var lastSampled *time.Time
	if !sampled.IsZero() {
		lastSampled = &sampled
	}
	return Latest{ServerID: id, Status: status, ObservedAt: received, LastSampledAt: lastSampled, MetricsStatus: metricsStatus, Stale: metricsStatus != "fresh" || status != "online", Metrics: items}, nil
}

func metricFreshness(observed, now time.Time, staleAfter time.Duration) string {
	if observed.IsZero() {
		return "missing"
	}
	if now.Sub(observed) > staleAfter {
		return "stale"
	}
	return "fresh"
}
func (s *Service) History(ctx context.Context, id, status, metricType, metricName, labelKey, labelValue, step string, from, to time.Time, limit int) (History, error) {
	if metricType != "cpu" && metricType != "memory" && metricType != "filesystem" && metricType != "network" {
		return History{}, ErrInvalid
	}
	if metricType == "filesystem" && labelKey != "mountpoint" {
		return History{}, ErrInvalid
	}
	if metricType == "network" && labelKey != "interface" {
		return History{}, ErrInvalid
	}
	if step == "raw" {
		items, err := s.repo.History(ctx, id, metricType, metricName, labelKey, labelValue, from, to, limit)
		if err != nil {
			return History{}, err
		}
		observed := historyObservedAt(items)
		freshness := metricFreshness(observed, time.Now().UTC(), s.staleAfter)
		return History{ServerID: id, MetricType: metricType, MetricName: metricName, Aggregation: "raw", MetricsStatus: freshness, Stale: freshness != "fresh" || status != "online", Series: items}, nil
	}
	stepSeconds := 0
	if step == "5m" {
		stepSeconds = 300
	} else if step == "1h" {
		stepSeconds = 3600
	} else {
		return History{}, ErrInvalid
	}
	items, err := s.repo.RollupHistory(ctx, id, metricType, metricName, labelKey, labelValue, from, to, stepSeconds, limit)
	if err != nil {
		return History{}, err
	}
	freshness := metricFreshness(historyObservedAt(items), time.Now().UTC(), s.staleAfter)
	missing, truncated := missingWindows(items, from, to, time.Duration(stepSeconds)*time.Second, limit)
	return History{ServerID: id, MetricType: metricType, MetricName: metricName, Aggregation: step, MetricsStatus: freshness, Stale: freshness != "fresh" || status != "online", Series: items, MissingWindows: missing, Truncated: truncated}, nil
}

// Missing windows are only asserted within the part of the result known to be
// complete. Cap enumeration just like returned data points.
func missingWindows(items []Point, from, to time.Time, step time.Duration, limit int) ([]time.Time, bool) {
	first := from.UTC().Truncate(step)
	if first.Before(from) {
		first = first.Add(step)
	}
	end := to.UTC()
	truncated := false
	if len(items) >= limit && len(items) > 0 {
		// Multiple labelled series can share a bucket, so don't assert the last
		// bucket is complete when the row limit was reached.
		end = items[len(items)-1].SampledAt.Add(-time.Nanosecond)
		truncated = true
	}
	seen := make(map[time.Time]bool, len(items))
	for _, item := range items {
		seen[item.SampledAt.UTC()] = true
	}
	missing := make([]time.Time, 0)
	for bucket, n := first, 0; !bucket.After(end); bucket, n = bucket.Add(step), n+1 {
		if n >= limit {
			truncated = true
			break
		}
		if !seen[bucket] {
			missing = append(missing, bucket)
		}
	}
	return missing, truncated
}

func historyObservedAt(items []Point) time.Time {
	var observed time.Time
	for _, item := range items {
		sampled := item.SampledAt
		if item.LastSampledAt != nil {
			sampled = *item.LastSampledAt
		}
		if sampled.After(observed) {
			observed = sampled
		}
	}
	return observed
}
func (s *Service) Cleanup(ctx context.Context) (int64, error) {
	removed, err := s.repo.Cleanup(ctx, time.Now().UTC().Add(-s.rawRetention), 500)
	if err != nil {
		return removed, err
	}
	rollupRemoved, err := s.repo.CleanupRollups(ctx, time.Now().UTC().Add(-s.rollup5mRetention), time.Now().UTC().Add(-s.rollup1hRetention), 500)
	return removed + rollupRemoved, err
}
func (s *Service) Aggregate(ctx context.Context, now time.Time) error {
	// Process bounded, bucket-aligned slices so repeated runs never overwrite a
	// complete bucket with a partial one. Revisit all available raw data,
	// including expired data after downtime, before allowing cleanup.
	oldest, err := s.repo.OldestSample(ctx)
	if err != nil {
		return err
	}
	if oldest.IsZero() {
		return nil
	}
	for _, step := range []int{300, 3600} {
		width := time.Duration(step) * time.Second
		start := oldest.UTC().Truncate(width)
		end := now.UTC().Truncate(width)
		for from := start; from.Before(end); {
			to := from.Add(24 * time.Hour)
			if to.After(end) {
				to = end
			}
			if err := s.repo.Aggregate(ctx, from, to, step); err != nil {
				return err
			}
			from = to
		}
	}
	return nil
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
				if err := s.Aggregate(ctx, time.Now().UTC()); err != nil {
					log.Printf("metrics aggregation failed: %v", err)
					continue
				}
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
