package database

import (
	"github.com/740297616/nexusops/server/migrations"
	"github.com/pressly/goose/v3"
)

func (d *Database) Migrations() (*goose.Provider, error) {
	return goose.NewProvider(goose.DialectPostgres, d.sql, migrations.Files)
}
