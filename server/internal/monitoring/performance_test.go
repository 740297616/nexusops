package monitoring

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/740297616/nexusops/server/internal/config"
	"github.com/740297616/nexusops/server/internal/database"
)

// TestMetricPerformance is opt-in because it writes a sizeable fixture to the
// database. Run it only against a disposable, already migrated database with
// NEXUSOPS_METRICS_PERF=1.
func TestMetricPerformance(t *testing.T) {
	if os.Getenv("NEXUSOPS_METRICS_PERF") != "1" {
		t.Skip("set NEXUSOPS_METRICS_PERF=1 with a disposable migrated database")
	}
	agents := perfIntEnv(t, "NEXUSOPS_METRICS_PERF_AGENTS", 4)
	samplesPerAgent := perfIntEnv(t, "NEXUSOPS_METRICS_PERF_SAMPLES", 1000)
	if agents < 1 || samplesPerAgent < 1 || agents > 32 || samplesPerAgent > 10000 {
		t.Fatal("performance fixture bounds are agents 1..32 and samples 1..10000")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(context.Background(), cfg.Database)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repo := NewRepository(db.GORM())
	ctx := context.Background()
	prefix := fmt.Sprintf("perf-%d-", time.Now().UnixNano())
	now := time.Now().UTC().Truncate(time.Second)
	serverIDs := make([]string, 0, agents)
	for agent := 0; agent < agents; agent++ {
		serverID := prefix + strconv.Itoa(agent)
		serverIDs = append(serverIDs, serverID)
		if err := db.GORM().Exec("INSERT INTO servers(id,name) VALUES(?,?)", serverID, serverID).Error; err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { _ = db.GORM().Exec("DELETE FROM servers WHERE id LIKE ?", prefix+"%").Error })

	var sizeBefore int64
	if err := db.GORM().Raw("SELECT pg_database_size(current_database())").Scan(&sizeBefore).Error; err != nil {
		t.Fatal(err)
	}
	insertStart := time.Now()
	for agent, serverID := range serverIDs {
		for sample := 0; sample < samplesPerAgent; sample++ {
			sampledAt := now.Add(-time.Duration(sample%samplesPerAgent) * time.Minute)
			requestID := fmt.Sprintf("%s%d-%d", prefix, agent, sample)
			result := db.GORM().Exec(`INSERT INTO metric_samples(server_id,request_id,schema_version,sampled_at,received_at) VALUES(?,?,?,?,?)`, serverID, requestID, 1, sampledAt, sampledAt.Add(time.Second))
			if result.Error != nil {
				t.Fatal(result.Error)
			}
			var sampleID int64
			if err := db.GORM().Raw("SELECT id FROM metric_samples WHERE server_id=? AND request_id=?", serverID, requestID).Scan(&sampleID).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.GORM().Exec("INSERT INTO metric_values(sample_id,metric_type,metric_name,value,unit) VALUES(?,?,?,?,?)", sampleID, "cpu", "usage_percent", float64(sample%100), "percent").Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	insertDuration := time.Since(insertStart)
	batchStart := time.Now()
	if err := db.GORM().Exec(`WITH inserted AS (
 INSERT INTO metric_samples(server_id,request_id,schema_version,sampled_at,received_at)
 SELECT ?, ? || n::text, 1, ?::timestamptz - n * interval '1 second', ?::timestamptz - n * interval '1 second'
 FROM generate_series(1, 1000) AS n RETURNING id
)
INSERT INTO metric_values(sample_id,metric_type,metric_name,value,unit)
SELECT id,'cpu','usage_percent',25,'percent' FROM inserted`, serverIDs[0], prefix+"batch-", now.Add(-48*time.Hour), now.Add(-48*time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	t.Logf("batch_insert=1000 %s", time.Since(batchStart))
	var sizeAfter int64
	if err := db.GORM().Raw("SELECT pg_database_size(current_database())").Scan(&sizeAfter).Error; err != nil {
		t.Fatal(err)
	}
	t.Logf("fixture agents=%d samples_per_agent=%d rows=%d sequential_insert=%s database_growth_bytes=%d", agents, samplesPerAgent, agents*samplesPerAgent, insertDuration, sizeAfter-sizeBefore)

	queryStart := time.Now()
	if _, err := repo.Latest(ctx, serverIDs[0], now, 2*time.Minute); err != nil {
		t.Fatal(err)
	}
	t.Logf("latest=%s", time.Since(queryStart))

	from, to := now.Add(-time.Duration(samplesPerAgent+1)*time.Minute), now.Add(time.Minute)
	queryStart = time.Now()
	if _, err := repo.History(ctx, serverIDs[0], "cpu", "usage_percent", "", "", from, to, 2000); err != nil {
		t.Fatal(err)
	}
	t.Logf("raw_history=%s", time.Since(queryStart))

	aggStart := time.Now()
	if err := repo.Aggregate(ctx, from, to, 300); err != nil {
		t.Fatal(err)
	}
	t.Logf("aggregate_5m=%s", time.Since(aggStart))
	var lastValue float64
	if err := db.GORM().Raw("SELECT last_value FROM metric_rollups WHERE server_id=? AND step_seconds=300 AND metric_type='cpu' AND metric_name='usage_percent' ORDER BY bucket_start DESC LIMIT 1", serverIDs[0]).Scan(&lastValue).Error; err != nil {
		t.Fatal(err)
	}
	if lastValue != 0 {
		t.Fatalf("rollup last_value = %v, want latest sample value 0", lastValue)
	}
	queryStart = time.Now()
	if _, err := repo.RollupHistory(ctx, serverIDs[0], "cpu", "usage_percent", "", "", from, to, 300, 2000); err != nil {
		t.Fatal(err)
	}
	t.Logf("rollup_5m_history=%s", time.Since(queryStart))

	aggStart = time.Now()
	if err := repo.Aggregate(ctx, from, to, 3600); err != nil {
		t.Fatal(err)
	}
	t.Logf("aggregate_1h=%s", time.Since(aggStart))
	queryStart = time.Now()
	if _, err := repo.RollupHistory(ctx, serverIDs[0], "cpu", "usage_percent", "", "", from, to, 3600, 2000); err != nil {
		t.Fatal(err)
	}
	t.Logf("rollup_1h_history=%s", time.Since(queryStart))
	var plan []string
	if err := db.GORM().Raw(`EXPLAIN SELECT * FROM metric_rollups WHERE server_id=? AND step_seconds=300 AND metric_type='cpu' AND metric_name='usage_percent' AND bucket_start>=? AND bucket_start<=? ORDER BY bucket_start LIMIT 2000`, serverIDs[0], from, to).Scan(&plan).Error; err != nil {
		t.Fatal(err)
	}
	for _, line := range plan {
		t.Logf("rollup_plan: %s", line)
	}
	plan = nil
	if err := db.GORM().Raw(`EXPLAIN SELECT * FROM metric_samples WHERE server_id=? AND sampled_at>=? AND sampled_at<=? ORDER BY sampled_at LIMIT 2000`, serverIDs[0], from, to).Scan(&plan).Error; err != nil {
		t.Fatal(err)
	}
	for _, line := range plan {
		t.Logf("raw_plan: %s", line)
	}

	// The entire database is disposable: cleanup intentionally tests broad retention deletion.
	cleanupStart := time.Now()
	if _, err := repo.CleanupRollups(ctx, now.Add(24*time.Hour), now.Add(24*time.Hour), 500); err != nil {
		t.Fatal(err)
	}
	t.Logf("rollup_cleanup=%s", time.Since(cleanupStart))
	cleanupStart = time.Now()
	if _, err := repo.Cleanup(ctx, now.Add(24*time.Hour), 500); err != nil {
		t.Fatal(err)
	}
	t.Logf("raw_cleanup=%s", time.Since(cleanupStart))
}

func perfIntEnv(t *testing.T, key string, fallback int) int {
	t.Helper()
	value := fallback
	if raw := os.Getenv(key); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			t.Fatalf("%s must be an integer: %v", key, err)
		}
		value = parsed
	}
	return value
}
