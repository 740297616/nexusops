COMPOSE = docker compose --env-file deploy/.env -f deploy/docker-compose.yml
DEV_COMPOSE = $(COMPOSE) -f deploy/docker-compose.dev.yml

.PHONY: help dev build up down logs migrate-up migrate-down agent-build

help:
	@echo "dev          Start Compose with Vite hot reload"
	@echo "build        Build production-style service images"
	@echo "up           Start production-style services"
	@echo "down         Stop services, preserving volumes"
	@echo "logs         Follow service logs"
	@echo "migrate-up   Apply database migrations"
	@echo "migrate-down Roll back the last migration"
	@echo "agent-build  Build Linux amd64 Agent"
	@echo "Create deploy/.env from deploy/.env.example first."

dev:
	$(DEV_COMPOSE) up -d --build

build:
	$(COMPOSE) build

up:
	$(COMPOSE) up -d --build

down:
	$(COMPOSE) down --remove-orphans

logs:
	$(COMPOSE) logs -f

migrate-up:
	$(COMPOSE) run --rm --build migrate up

migrate-down:
	$(COMPOSE) run --rm --build migrate down

agent-build:
	cd agent && GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=0.1.0" -o nexusops-agent ./cmd/agent
