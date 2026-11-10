FROM golang:1.24.10-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/server ./cmd/server \
    && CGO_ENABLED=0 go build -trimpath -o /out/migrate ./cmd/migrate \
    && CGO_ENABLED=0 go build -trimpath -o /out/init-admin ./cmd/init-admin

FROM debian:bookworm-slim AS runtime
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/server /usr/local/bin/nexusops-server
COPY --from=build /out/migrate /usr/local/bin/nexusops-migrate
COPY --from=build /out/init-admin /usr/local/bin/nexusops-init-admin
USER 65532:65532
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 CMD ["curl", "--fail", "--silent", "--max-time", "3", "http://127.0.0.1:8080/api/v1/ready"]
ENTRYPOINT ["/usr/local/bin/nexusops-server"]
