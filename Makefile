# Команды локальной разработки (Git Bash / WSL / Make for Windows).
# Эквиваленты для PowerShell — в docs/local-development.md

COMPOSE := docker compose
ENV_FILE := .env

.PHONY: help env-init up up-build down stop logs logs-backend logs-frontend logs-postgres \
	ps migrate-up migrate-down migrate-version db-reset test test-backend test-frontend \
	config

help:
	@echo "Targets:"
	@echo "  env-init         - copy .env.example -> .env if missing"
	@echo "  config           - validate docker compose config"
	@echo "  up               - start stack (detached)"
	@echo "  up-build         - rebuild images and start"
	@echo "  stop             - stop containers, keep volumes/data"
	@echo "  down             - remove containers/network, keep volumes"
	@echo "  logs             - follow all logs"
	@echo "  migrate-up       - apply DB migrations"
	@echo "  migrate-down     - roll back one migration"
	@echo "  migrate-version  - show migration version"
	@echo "  db-reset         - DESTRUCTIVE: down -v and recreate + migrate"
	@echo "  test             - run backend + frontend tests"
	@echo "  test-backend     - go test ./..."
	@echo "  test-frontend    - npm test (vitest)"

env-init:
	@if [ ! -f $(ENV_FILE) ]; then cp .env.example $(ENV_FILE); echo "Created $(ENV_FILE)"; else echo "$(ENV_FILE) already exists"; fi

config:
	$(COMPOSE) config

up: env-init
	$(COMPOSE) up -d

up-build: env-init
	$(COMPOSE) up -d --build

stop:
	$(COMPOSE) stop

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f --tail=200

logs-backend:
	$(COMPOSE) logs -f --tail=200 backend

logs-frontend:
	$(COMPOSE) logs -f --tail=200 frontend

logs-postgres:
	$(COMPOSE) logs -f --tail=200 postgres

ps:
	$(COMPOSE) ps

migrate-up: env-init
	$(COMPOSE) --profile tools run --rm migrate

migrate-down: env-init
	$(COMPOSE) --profile tools run --rm --entrypoint /bin/sh migrate -c 'migrate -path=/migrations -database "$$DATABASE_URL" down 1'

migrate-version: env-init
	$(COMPOSE) --profile tools run --rm --entrypoint /bin/sh migrate -c 'migrate -path=/migrations -database "$$DATABASE_URL" version'

# Явно разрушающая команда: удаляет volume PostgreSQL и поднимает стек заново.
db-reset:
	@echo "WARNING: This deletes local PostgreSQL volume data."
	$(COMPOSE) down -v
	$(COMPOSE) up -d --build
	@$(MAKE) migrate-up

test: test-backend test-frontend

test-backend:
	$(COMPOSE) exec -T backend go test ./...

test-frontend:
	$(COMPOSE) exec -T frontend npm test

config-check: config
