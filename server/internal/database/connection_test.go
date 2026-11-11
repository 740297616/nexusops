package database

import (
	"context"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/740297616/nexusops/server/internal/config"
	"github.com/pressly/goose/v3"
	"gorm.io/gorm"
)

func TestConnectionFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	cfg := config.Database{Host: "127.0.0.1", Port: port, User: "test", Password: "test", Name: "test", SSLMode: "disable", MaxOpenConns: 1, MaxIdleConns: 0, ConnMaxLifetime: time.Minute, ConnectTimeout: time.Second}
	if _, err := Open(context.Background(), cfg); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected unavailable, got %v", err)
	}
}

// This test changes Goose migration state; use an isolated disposable database only.
func TestPostgresIntegration(t *testing.T) {
	if os.Getenv("NEXUSOPS_DB_INTEGRATION") != "1" {
		t.Skip("set NEXUSOPS_DB_INTEGRATION=1 with a disposable database")
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Database.MaxOpenConns = 2
	cfg.Database.MaxIdleConns = 1
	cfg.Database.ConnMaxLifetime = 100 * time.Millisecond
	db, err := Open(context.Background(), cfg.Database)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	if err := db.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	if db.sql.Stats().MaxOpenConnections != cfg.Database.MaxOpenConns {
		t.Fatal("max connections not applied")
	}
	first, err := db.sql.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	second, err := db.sql.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
	if db.sql.Stats().Idle > 1 || db.sql.Stats().MaxIdleClosed == 0 {
		t.Fatal("max idle connections not applied")
	}
	time.Sleep(150 * time.Millisecond)
	if err := db.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	if db.sql.Stats().MaxLifetimeClosed == 0 {
		t.Fatal("connection lifetime not applied")
	}
	sentinel := errors.New("rollback")
	if err := db.Transaction(ctx, func(tx *gorm.DB) error {
		if err := tx.Exec("CREATE TEMP TABLE nexusops_transaction_test (value integer)").Error; err != nil {
			return err
		}
		return sentinel
	}); !errors.Is(err, sentinel) {
		t.Fatalf("transaction error: %v", err)
	}
	var relation *string
	if err := db.GORM().Raw("SELECT to_regclass('pg_temp.nexusops_transaction_test')::text").Scan(&relation).Error; err != nil || relation != nil {
		t.Fatalf("transaction did not roll back: %v", err)
	}
	provider, err := db.Migrations()
	if err != nil {
		t.Fatal(err)
	}
	statuses, err := provider.Status(ctx)
	if err != nil || len(statuses) != 7 || !allMigrationState(statuses, goose.StatePending) {
		t.Fatalf("expected a fresh isolated database: %v %v", statuses, err)
	}
	results, err := provider.Up(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 7 {
		t.Fatalf("expected fresh isolated database, got %d migrations", len(results))
	}
	results, err = provider.Up(ctx)
	if err != nil || len(results) != 0 {
		t.Fatalf("repeated up: %v %v", results, err)
	}
	statuses, err = provider.Status(ctx)
	if err != nil || len(statuses) != 7 || !allMigrationState(statuses, goose.StateApplied) {
		t.Fatalf("status: %v %v", statuses, err)
	}
	if _, err := provider.Down(ctx); err != nil {
		t.Fatal(err)
	}
	statuses, err = provider.Status(ctx)
	if err != nil || len(statuses) != 7 || statuses[6].State != goose.StatePending || !allMigrationState(statuses[:6], goose.StateApplied) {
		t.Fatalf("down status: %v %v", statuses, err)
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(db.Ping(ctx), ErrUnavailable) {
		t.Fatal("closed database still responds")
	}
}

func allMigrationState(statuses []*goose.MigrationStatus, state goose.State) bool {
	for _, status := range statuses {
		if status.State != state {
			return false
		}
	}
	return true
}
