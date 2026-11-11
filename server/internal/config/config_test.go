package config

import (
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	for key, value := range map[string]string{"DB_HOST": "localhost", "DB_PORT": "5433", "DB_USER": "user", "DB_PASSWORD": "secret", "DB_NAME": "testdb", "DB_SSLMODE": "require", "DB_MAX_OPEN_CONNS": "8", "DB_MAX_IDLE_CONNS": "3", "DB_CONN_MAX_LIFETIME": "2m", "DB_CONNECT_TIMEOUT": "3s"} {
		t.Setenv(key, value)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	db := cfg.Database
	if db.Host != "localhost" || db.Port != 5433 || db.User != "user" || db.Password != "secret" || db.Name != "testdb" || db.SSLMode != "require" || db.MaxOpenConns != 8 || db.MaxIdleConns != 3 || db.ConnMaxLifetime != 2*time.Minute || db.ConnectTimeout != 3*time.Second {
		t.Fatal("environment settings were not loaded")
	}
}

func TestInvalidConfig(t *testing.T) {
	valid := Database{Host: "localhost", Port: 5432, User: "user", Password: "secret", Name: "db", SSLMode: "require", MaxOpenConns: 5, MaxIdleConns: 2, ConnMaxLifetime: time.Minute, ConnectTimeout: time.Second}
	tests := []struct {
		name   string
		change func(*Database)
	}{
		{"missing password", func(d *Database) { d.Password = "" }}, {"bad port", func(d *Database) { d.Port = 0 }}, {"bad pool", func(d *Database) { d.MaxIdleConns = 6 }}, {"bad lifetime", func(d *Database) { d.ConnMaxLifetime = 0 }}, {"bad timeout", func(d *Database) { d.ConnectTimeout = 0 }}, {"bad sslmode", func(d *Database) { d.SSLMode = "prefer" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := valid
			tc.change(&db)
			if db.Validate() == nil {
				t.Fatal("expected validation error")
			}
		})
	}
	t.Setenv("DB_PORT", "bad")
	if _, err := Load(); err == nil {
		t.Fatal("invalid DB_PORT accepted")
	}
}

func TestSecurityConfig(t *testing.T) {
	for key, value := range map[string]string{"DB_USER": "user", "DB_PASSWORD": "password", "DB_NAME": "db", "NEXUSOPS_TRUSTED_PROXY_CIDRS": "10.0.0.0/8, 192.168.0.0/16", "AUTH_RATE_LIMIT_IP_MAX": "3", "AUTH_RATE_LIMIT_ACCOUNT_MAX": "2", "AUTH_RATE_LIMIT_WINDOW": "2m", "AUTH_CLEANUP_INTERVAL": "5m", "AUTH_SESSION_RETENTION": "1h", "AUTH_AUDIT_RETENTION": "24h"} {
		t.Setenv(key, value)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Security.TrustedProxyCIDRs) != 2 || cfg.Security.RateLimitIPMax != 3 || cfg.Security.RateLimitAccountMax != 2 || cfg.Security.RateLimitWindow != 2*time.Minute || cfg.Security.CleanupInterval != 5*time.Minute || cfg.Security.SessionRetention != time.Hour || cfg.Security.AuditRetention != 24*time.Hour {
		t.Fatalf("unexpected security config: %+v", cfg.Security)
	}
}
