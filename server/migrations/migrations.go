package migrations

import "embed"

// Files contains the migrations shipped with the executable.
//
//go:embed *.sql
var Files embed.FS
