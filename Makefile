SHELL := /bin/bash
GO ?= go
DATA_DIR ?= .data
PKGS := ./...

.DEFAULT_GOAL := help

.PHONY: help dev down test lint proto build seed load migrate demo refresh-pricing run-ingest run-api run-processor run-inference run-cost-api test-integration docker-config clean

help: ## Show available targets.
	@awk 'BEGIN {FS = ":.*##"; printf "Targets:\n"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

dev: ## Start local dependencies with docker compose when Docker is available.
	@docker compose up -d postgres localstack prometheus grafana tempo || echo "docker compose unavailable; local file-backed services still work"

down: ## Stop local dependencies.
	@docker compose down || true

test: ## Run Go tests.
	$(GO) test $(PKGS)

lint: ## Run lightweight local lint checks.
	@files="$$(find . -path './web/cost/node_modules' -prune -o -path './.git' -prune -o -name '*.go' -print)"; \
	unformatted="$$(gofmt -l $$files)"; \
	if [ -n "$$unformatted" ]; then printf '%s\n' "$$unformatted"; exit 1; fi
	$(GO) test $(PKGS)
	@if command -v terraform >/dev/null 2>&1 && [ -d infra ]; then terraform fmt -recursive -check infra; fi

proto: ## Regenerate committed proto stubs and OpenAPI snapshot.
	$(GO) run ./scripts/genproto

build: proto ## Build all Go command binaries.
	$(GO) build ./cmd/...

seed: ## Seed demo user/device into the local JSON store.
	THRONE_DATA_DIR=$(DATA_DIR) $(GO) run ./scripts/seed

migrate: ## Apply local SQL migrations marker for demo/dev.
	THRONE_DATA_DIR=$(DATA_DIR) $(GO) run ./scripts/migrate

load: ## Run the configurable load generator; pass flags after --.
	THRONE_DATA_DIR=$(DATA_DIR) $(GO) run ./cmd/throne-load --target http://localhost:8080 $(filter-out $@,$(MAKECMDGOALS))

run-ingest: ## Run the ingest HTTP/gRPC-shaped service.
	THRONE_DATA_DIR=$(DATA_DIR) $(GO) run ./cmd/throne-ingest

run-api: ## Run the mobile-facing API service.
	THRONE_DATA_DIR=$(DATA_DIR) $(GO) run ./cmd/throne-api

run-processor: ## Run one processor drain loop.
	THRONE_DATA_DIR=$(DATA_DIR) $(GO) run ./cmd/throne-processor --once

run-inference: ## Run the inference stub service.
	$(GO) run ./cmd/throne-inference

run-cost-api: ## Run the AWS-pricing-backed cost API with snapshot fallback.
	THRONE_DATA_DIR=$(DATA_DIR) $(GO) run ./cmd/throne-cost-api

refresh-pricing: ## Refresh the bundled pricing snapshot from current fallback/live constants.
	THRONE_DATA_DIR=$(DATA_DIR) $(GO) run ./cmd/throne-cost-api --refresh-pricing

test-integration: ## Run local integration-style Go tests.
	$(GO) test ./test/integration -count=1

docker-config: ## Validate docker-compose syntax.
	docker compose config >/dev/null

demo: ## Bring up dependencies, seed data, generate load, process it, and print demo URLs.
	./scripts/demo.sh

clean: ## Remove local runtime data.
	rm -rf $(DATA_DIR) .cache

%:
	@:
