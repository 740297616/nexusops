# NexusOps Server

Go 1.24 or newer is required. Run `go run ./cmd/server` from this directory. The default listener is `:8080`; override it with `NEXUSOPS_SERVER_ADDR`. `NEXUSOPS_VERSION` sets the version returned by `GET /api/v1/health` (default `0.1.0`).

The endpoint reports process availability only; it does not check a database or other dependencies. Run `go test ./...` and `go vet ./...` for verification. TLS must be terminated by a trusted proxy for non-local use.
