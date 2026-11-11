package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type Database struct {
	Host, User, Password, Name, SSLMode string
	Port, MaxOpenConns, MaxIdleConns    int
	ConnMaxLifetime, ConnectTimeout     time.Duration
}

type Config struct {
	Address, Version string
	Database         Database
	Security         Security
}

type Security struct {
	TrustedProxyCIDRs                                                                   []string
	RateLimitIPMax, RateLimitAccountMax                                                 int
	RateLimitWindow, CleanupInterval, SessionRetention, AuditRetention                  time.Duration
	AgentHeartbeatTimeout, AgentRegistrationTTL                                         time.Duration
	AgentRateLimitWindow                                                                time.Duration
	AgentRegisterRateLimit, AgentHeartbeatRateLimit, AgentReportRateLimit               int
	AgentMetricsRateLimit                                                               int
	MetricsCleanupInterval, MetricsRawRetention, Metrics5mRetention, Metrics1hRetention time.Duration
	MetricsStaleAfter                                                                   time.Duration
}

func Load() (Config, error) {
	cfg := Config{Address: envOr("NEXUSOPS_SERVER_ADDR", ":8080"), Version: envOr("NEXUSOPS_VERSION", "0.1.0")}
	db := &cfg.Database
	db.Host = envOr("DB_HOST", "127.0.0.1")
	db.User, db.Password, db.Name = os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME")
	db.SSLMode = envOr("DB_SSLMODE", "verify-full")
	var err error
	if db.Port, err = intEnv("DB_PORT", "5432"); err != nil {
		return Config{}, err
	}
	if db.MaxOpenConns, err = intEnv("DB_MAX_OPEN_CONNS", "10"); err != nil {
		return Config{}, err
	}
	if db.MaxIdleConns, err = intEnv("DB_MAX_IDLE_CONNS", "5"); err != nil {
		return Config{}, err
	}
	if db.ConnMaxLifetime, err = durationEnv("DB_CONN_MAX_LIFETIME", "30m"); err != nil {
		return Config{}, err
	}
	if db.ConnectTimeout, err = durationEnv("DB_CONNECT_TIMEOUT", "5s"); err != nil {
		return Config{}, err
	}
	if err := db.Validate(); err != nil {
		return Config{}, err
	}
	security := Security{TrustedProxyCIDRs: splitEnv("NEXUSOPS_TRUSTED_PROXY_CIDRS"), RateLimitIPMax: 20, RateLimitAccountMax: 10}
	if security.RateLimitIPMax, err = intEnv("AUTH_RATE_LIMIT_IP_MAX", "20"); err != nil {
		return Config{}, err
	}
	if security.RateLimitAccountMax, err = intEnv("AUTH_RATE_LIMIT_ACCOUNT_MAX", "10"); err != nil {
		return Config{}, err
	}
	if security.RateLimitWindow, err = durationEnv("AUTH_RATE_LIMIT_WINDOW", "1m"); err != nil {
		return Config{}, err
	}
	if security.CleanupInterval, err = durationEnv("AUTH_CLEANUP_INTERVAL", "1h"); err != nil {
		return Config{}, err
	}
	if security.SessionRetention, err = durationEnv("AUTH_SESSION_RETENTION", "24h"); err != nil {
		return Config{}, err
	}
	if security.AuditRetention, err = durationEnv("AUTH_AUDIT_RETENTION", "2160h"); err != nil {
		return Config{}, err
	}
	if security.AgentHeartbeatTimeout, err = durationEnv("AGENT_HEARTBEAT_TIMEOUT", "2m"); err != nil {
		return Config{}, err
	}
	if security.AgentRegistrationTTL, err = durationEnv("AGENT_REGISTRATION_TTL", "15m"); err != nil {
		return Config{}, err
	}
	if security.AgentRateLimitWindow, err = durationEnv("AGENT_RATE_LIMIT_WINDOW", "1m"); err != nil {
		return Config{}, err
	}
	if security.AgentRegisterRateLimit, err = intEnv("AGENT_REGISTER_RATE_LIMIT", "10"); err != nil {
		return Config{}, err
	}
	if security.AgentHeartbeatRateLimit, err = intEnv("AGENT_HEARTBEAT_RATE_LIMIT", "120"); err != nil {
		return Config{}, err
	}
	if security.AgentReportRateLimit, err = intEnv("AGENT_REPORT_RATE_LIMIT", "30"); err != nil {
		return Config{}, err
	}
	if security.AgentMetricsRateLimit, err = intEnv("AGENT_METRICS_RATE_LIMIT", "60"); err != nil {
		return Config{}, err
	}
	if security.MetricsCleanupInterval, err = durationEnv("METRICS_CLEANUP_INTERVAL", "1h"); err != nil {
		return Config{}, err
	}
	if security.MetricsStaleAfter, err = durationEnv("METRICS_STALE_AFTER", "2m"); err != nil {
		return Config{}, err
	}
	if security.MetricsRawRetention, err = durationEnv("METRICS_RAW_RETENTION", "168h"); err != nil {
		return Config{}, err
	}
	if security.Metrics5mRetention, err = durationEnv("METRICS_5M_RETENTION", "720h"); err != nil {
		return Config{}, err
	}
	if security.Metrics1hRetention, err = durationEnv("METRICS_1H_RETENTION", "4320h"); err != nil {
		return Config{}, err
	}
	if security.RateLimitIPMax < 1 || security.RateLimitAccountMax < 1 || security.RateLimitWindow <= 0 || security.CleanupInterval <= 0 || security.SessionRetention < 0 || security.AuditRetention < 0 || security.AgentHeartbeatTimeout <= 0 || security.AgentRegistrationTTL <= 0 || security.AgentRateLimitWindow <= 0 || security.AgentRegisterRateLimit < 1 || security.AgentHeartbeatRateLimit < 1 || security.AgentReportRateLimit < 1 || security.AgentMetricsRateLimit < 1 || security.MetricsCleanupInterval <= 0 || security.MetricsStaleAfter <= 0 || security.MetricsRawRetention <= 0 || security.Metrics5mRetention <= 0 || security.Metrics1hRetention <= 0 {
		return Config{}, fmt.Errorf("security limits and cleanup durations must be valid")
	}
	cfg.Security = security
	return cfg, nil
}

func (db Database) Validate() error {
	if strings.TrimSpace(db.Host) == "" || strings.TrimSpace(db.User) == "" || db.Password == "" || strings.TrimSpace(db.Name) == "" {
		return fmt.Errorf("DB_HOST, DB_USER, DB_PASSWORD and DB_NAME are required")
	}
	if db.Port < 1 || db.Port > 65535 {
		return fmt.Errorf("DB_PORT must be between 1 and 65535")
	}
	if db.MaxOpenConns < 1 || db.MaxIdleConns < 0 || db.MaxIdleConns > db.MaxOpenConns {
		return fmt.Errorf("DB_MAX_OPEN_CONNS must be positive and DB_MAX_IDLE_CONNS must be between 0 and DB_MAX_OPEN_CONNS")
	}
	if db.ConnMaxLifetime <= 0 || db.ConnectTimeout <= 0 {
		return fmt.Errorf("DB_CONN_MAX_LIFETIME and DB_CONNECT_TIMEOUT must be positive")
	}
	switch db.SSLMode {
	case "disable", "require", "verify-ca", "verify-full":
	default:
		return fmt.Errorf("DB_SSLMODE must be disable, require, verify-ca or verify-full")
	}
	return nil
}

func (db Database) Address() string { return net.JoinHostPort(db.Host, strconv.Itoa(db.Port)) }
func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func intEnv(key, fallback string) (int, error) {
	value, err := strconv.Atoi(envOr(key, fallback))
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", key)
	}
	return value, nil
}
func durationEnv(key, fallback string) (time.Duration, error) {
	value, err := time.ParseDuration(envOr(key, fallback))
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration such as 5s or 30m", key)
	}
	return value, nil
}
func splitEnv(key string) []string {
	parts := strings.Split(os.Getenv(key), ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			values = append(values, value)
		}
	}
	return values
}
