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
