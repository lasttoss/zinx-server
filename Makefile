SHELL := /bin/bash
GO ?= go
COMPOSE ?= docker compose

.PHONY: help up down logs restart infra build run test race vet fmt lint coverage smoke clean diagram

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

up: ## Start the whole stack (mongodb + redis + server) and smoke test it
	$(COMPOSE) up -d --build
	@$(MAKE) --no-print-directory smoke

down: ## Stop the stack
	$(COMPOSE) down

logs: ## Tail logs
	$(COMPOSE) logs -f zinx-server

infra: ## Start only mongodb + redis (for local `go run`)
	$(COMPOSE) up -d mongodb redis

build: ## Build the binary
	$(GO) build -o bin/server ./cmd/server

run: ## Run locally (needs `make infra`)
	$(GO) run ./cmd/server

test: ## Unit tests
	$(GO) test ./...

race: ## Unit tests with the race detector
	$(GO) test -race ./...

vet: ## Static analysis
	$(GO) vet ./...

fmt: ## Format
	$(GO) fmt ./...

coverage: ## Coverage report
	$(GO) test -coverprofile=coverage.out ./... && $(GO) tool cover -func=coverage.out | tail -1

lint: fmt vet ## Format + vet

smoke: ## End-to-end smoke test against a running server
	@PORT=$$(grep -E '^SERVER_PORT=' .env 2>/dev/null | cut -d= -f2); \
	python3 scripts/smoke.py 127.0.0.1 $${PORT:-8999}

clean: ## Remove build artifacts
	rm -rf bin coverage.out

# Sources are HTML and Mermaid; a PNG is a build artifact.
diagram:
	@if command -v chromium >/dev/null 2>&1; then B=chromium; elif command -v google-chrome >/dev/null 2>&1; then B=google-chrome; else echo "no chromium on PATH: open docs/diagrams/*.html in a browser"; exit 0; fi; \
	for f in docs/diagrams/*.html; do $$B --headless --screenshot="$${f%.html}.png" --window-size=1200,1000 "$$f" && echo "wrote $${f%.html}.png"; done
