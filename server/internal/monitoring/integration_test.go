package monitoring

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/740297616/nexusops/server/internal/config"
	"github.com/740297616/nexusops/server/internal/database"
	"github.com/740297616/nexusops/server/internal/serverasset"
)

// The database must be migrated and disposable: retention cleanup is global.
func TestMonitoringPostgres(t *testing.T) {
	if os.Getenv("NEXUSOPS_METRICS_INTEGRATION") != "1" {
		t.Skip("set NEXUSOPS_METRICS_INTEGRATION=1 with a disposable migrated database")
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
	ctx := context.Background()
	repo := NewRepository(db.GORM())
	id := fmt.Sprintf("test-%d", time.Now().UnixNano())
	if err := db.GORM().Exec("INSERT INTO servers(id,name) VALUES(?,?)", id, id).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.GORM().Exec("DELETE FROM servers WHERE id=?", id).Error })
	var credentialID int64
	if err := db.GORM().Raw("INSERT INTO agent_credentials(server_id,credential_hash) VALUES(?,?) RETURNING id", id, []byte(id)).Scan(&credentialID).Error; err != nil {
		t.Fatal(err)
	}
	service := NewService(repo, time.Minute, 7*24*time.Hour, 30*24*time.Hour, 180*24*time.Hour)
	latest, err := service.Latest(ctx, id, "online")
	if err != nil || latest.MetricsStatus != "missing" || latest.LastSampledAt != nil {
		t.Fatalf("missing: %+v %v", latest, err)
	}
	base := time.Now().UTC().Truncate(time.Hour).Add(-2 * time.Hour)
	insert := func(request string, at time.Time, cpu, network float64) {
		t.Helper()
		_, err := repo.Insert(ctx, serverasset.AgentPrincipal{ServerID: id, CredentialID: credentialID}, IngestRequest{RequestID: request, SchemaVersion: 1, SampledAt: at, Samples: []Sample{
			{MetricType: "cpu", MetricName: "usage_percent", Value: cpu, Unit: "percent"},
			{MetricType: "network", MetricName: "rx_bytes_total", Labels: map[string]string{"interface": "eth0"}, Value: network, Unit: "bytes"},
			{MetricType: "filesystem", MetricName: "usage_percent", Labels: map[string]string{"mountpoint": "/", "filesystem": "ext4"}, Value: cpu, Unit: "percent"},
		}}, at)
		if err != nil {
			t.Fatal(err)
		}
	}
	insert("first", base.Add(time.Minute), 80, 500)
	insert("second", base.Add(2*time.Minute), 20, 100)
	for i := 0; i < 2; i++ {
		if err := repo.Aggregate(ctx, base, base.Add(time.Hour), 300); err != nil {
			t.Fatal(err)
		}
	}
	points, err := repo.RollupHistory(ctx, id, "cpu", "usage_percent", "", "", base, base.Add(time.Hour), 300, 100)
	if err != nil || len(points) != 1 || points[0].Value != 50 || *points[0].MinValue != 20 || *points[0].MaxValue != 80 || *points[0].SampleCount != 2 {
		t.Fatalf("cpu rollup: %+v %v", points, err)
	}
	network, err := repo.RollupHistory(ctx, id, "network", "rx_bytes_total", "interface", "eth0", base, base.Add(time.Hour), 300, 100)
	if err != nil || len(network) != 1 || network[0].Value != 100 || network[0].Labels["interface"] != "eth0" {
		t.Fatalf("network latest counter: %+v %v", network, err)
	}
	raw, err := repo.History(ctx, id, "filesystem", "usage_percent", "mountpoint", "/", base, base.Add(time.Hour), 100)
	if err != nil || len(raw) != 2 || raw[0].Labels["filesystem"] != "ext4" {
		t.Fatalf("raw filesystem: %+v %v", raw, err)
	}
	insert("late", base.Add(3*time.Minute), 50, 150)
	if err := repo.Aggregate(ctx, base, base.Add(time.Hour), 300); err != nil {
		t.Fatal(err)
	}
	points, err = repo.RollupHistory(ctx, id, "cpu", "usage_percent", "", "", base, base.Add(time.Hour), 300, 100)
	if err != nil || len(points) != 1 || points[0].Value != 50 || *points[0].SampleCount != 3 {
		t.Fatalf("late sample: %+v %v", points, err)
	}
	// A concurrent or later partial scan must not overwrite a complete bucket.
	if err := repo.Aggregate(ctx, base.Add(2*time.Minute), base.Add(5*time.Minute), 300); err != nil {
		t.Fatal(err)
	}
	points, err = repo.RollupHistory(ctx, id, "cpu", "usage_percent", "", "", base, base.Add(time.Hour), 300, 100)
	if err != nil || len(points) != 1 || *points[0].SampleCount != 3 {
		t.Fatalf("partial rollup replaced complete bucket: %+v %v", points, err)
	}
	// Simulate downtime longer than the raw retention: still-available expired
	// samples must be rolled up before cleanup.
	if err := service.Aggregate(ctx, base.Add(8*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	hourly, err := repo.RollupHistory(ctx, id, "cpu", "usage_percent", "", "", base, base.Add(time.Hour), 3600, 100)
	if err != nil || len(hourly) != 1 || *hourly[0].SampleCount != 3 {
		t.Fatalf("expired raw backfill: %+v %v", hourly, err)
	}
	latest, err = service.Latest(ctx, id, "offline")
	if err != nil || latest.MetricsStatus != "stale" || latest.Status != "offline" || latest.LastSampledAt == nil {
		t.Fatalf("offline latest: %+v %v", latest, err)
	}
	if _, err := repo.Cleanup(ctx, base, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CleanupRollups(ctx, base.Add(4*time.Minute), base.Add(59*time.Minute), 2); err != nil {
		t.Fatal(err)
	}
	points, err = repo.RollupHistory(ctx, id, "cpu", "usage_percent", "", "", base, base.Add(time.Hour), 300, 100)
	if err != nil || len(points) != 1 {
		t.Fatalf("within retention removed: %+v %v", points, err)
	}
	if _, err := repo.CleanupRollups(ctx, base.Add(5*time.Minute), base.Add(time.Hour), 2); err != nil {
		t.Fatal(err)
	}
	if n, err := repo.CleanupRollups(ctx, base.Add(5*time.Minute), base.Add(time.Hour), 2); err != nil || n != 0 {
		t.Fatalf("repeat cleanup: %d %v", n, err)
	}
}
