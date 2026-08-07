# Database Makefile
# Requires: goose CLI (go install github.com/pressly/goose/v3/cmd/goose@latest)
# Requires: Docker (for test target)

# Auto-load environment variables from .env if present
# TODO: This loads ./.env relative to the database/ directory, but project env vars
# are currently stored in server/.env. Update this to point to the correct env file
# or standardize on a single project-root .env so `make up` works from database/.
-include ./.env

DB_URL ?= postgresql://$(DB_USERNAME):$(DB_PASSWORD)@$(DB_IP)/$(DB_NAME)?sslmode=disable

.PHONY: create up down status version test

create: ## Create a new migration: make create name=add_teams
	goose -dir ./migrations create -s $(name) sql

up: ## Run all pending migrations
	@echo "Running migrations..."
	@goose -dir ./migrations postgres "$(DB_URL)" up

down: ## Rollback one migration
	@echo "Rolling back one migration..."
	@goose -dir ./migrations postgres "$(DB_URL)" down

status: ## Show migration status
	@goose -dir ./migrations postgres "$(DB_URL)" status

version: ## Show current migration version
	@goose -dir ./migrations postgres "$(DB_URL)" version

test: ## Test all migrations in Docker (full up/down cycle)
	./test-migrations.sh
