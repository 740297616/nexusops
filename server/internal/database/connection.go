package database

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"time"

	"github.com/740297616/nexusops/server/internal/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var ErrUnavailable = errors.New("database unavailable")

type Database struct {
	db      *gorm.DB
	sql     *sql.DB
	timeout time.Duration
}

func Open(ctx context.Context, cfg config.Database) (*Database, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	u := &url.URL{Scheme: "postgres", User: url.UserPassword(cfg.User, cfg.Password), Host: cfg.Address(), Path: "/" + cfg.Name}
	query := u.Query()
	query.Set("sslmode", cfg.SSLMode)
	u.RawQuery = query.Encode()
	pgcfg, err := pgx.ParseConfig(u.String())
	if err != nil {
		return nil, errors.New("invalid database connection configuration")
	}
	pgcfg.ConnectTimeout = cfg.ConnectTimeout
	sqlDB := sql.OpenDB(stdlib.GetConnector(*pgcfg))
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		_ = sqlDB.Close()
		return nil, ErrUnavailable
	}
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{DisableAutomaticPing: true, TranslateError: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		_ = sqlDB.Close()
		return nil, errors.New("database initialization failed")
	}
	return &Database{db: db, sql: sqlDB, timeout: cfg.ConnectTimeout}, nil
}

func (d *Database) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()
	if err := d.sql.PingContext(ctx); err != nil {
		return ErrUnavailable
	}
	return nil
}

// GORM returns the shared handle for future repositories; it never opens a new pool.
func (d *Database) GORM() *gorm.DB { return d.db }
func (d *Database) Transaction(ctx context.Context, fn func(*gorm.DB) error) error {
	return d.db.WithContext(ctx).Transaction(fn)
}
func (d *Database) Close() error { return d.sql.Close() }
