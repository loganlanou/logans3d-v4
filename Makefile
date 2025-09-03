SHELL := /bin/bash

# Go environment
export GOROOT=~/go-install/go
export PATH=$(GOROOT)/bin:$(PATH)
export GOPATH=~/go

.PHONY: install-tools
install-tools:
	go install github.com/a-h/templ/cmd/templ@latest
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

.PHONY: templ-update
templ-update:
	go install github.com/a-h/templ/cmd/templ@latest
	go get -u github.com/a-h/templ@latest

.PHONY: generate
generate:
	go generate ./...

.PHONY: dev
dev:
	@echo "🚀 Starting development server with Air (hot reload + auto-regeneration)..."
	air

.PHONY: run
run:
	@echo "⚠️  WARNING: Use 'air' for development instead of 'make run'"
	@echo "   Air handles auto-regeneration and hot reloading automatically"
	@echo "   Run 'air' directly or 'make dev' (which calls air)"
	@exit 1

.PHONY: build
build:
	go build -o logans3d ./cmd

.PHONY: test
test:
	go test ./... || echo "No tests found. Skipping."

.PHONY: lint
lint:
	golangci-lint run --max-issues-per-linter=0 || echo "Linter not installed. Run make install-linter"

.PHONY: install-linter
install-linter:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v2.1.6

.PHONY: migrate
migrate:
	mkdir -p db && goose -dir storage/migrations sqlite3 ./db/logans3d.db up

.PHONY: migrate-down
migrate-down:
	goose -dir storage/migrations sqlite3 ./db/logans3d.db down

.PHONY: migrate-status
migrate-status:
	goose -dir storage/migrations sqlite3 ./db/logans3d.db status

.PHONY: sqlc-generate
sqlc-generate:
	sqlc generate -f storage/sqlc.yaml

.PHONY: seed
seed:
	go run scripts/seed-db/main.go -db db/logans3d.db

.PHONY: css
css:
	postcss public/css/input.css -o public/css/styles.css

.PHONY: css-watch
css-watch:
	postcss public/css/input.css -o public/css/styles.css --watch

.PHONY: images
images:
	go run scripts/image-process/main.go

.PHONY: e2e
e2e:
	npm test

.PHONY: e2e-ui
e2e-ui:
	npm run test:ui

.PHONY: e2e-headed
e2e-headed:
	npm run test:headed

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: clean
clean:
	rm -f logans3d
	rm -rf tmp/
	rm -rf storage/db/
	rm -rf node_modules/.playwright

.PHONY: setup
setup: install-tools tidy
	npm install
	mkdir -p db
	make migrate
	make css
	@echo "Setup complete! Run 'air' to start development server (with auto-regeneration)"

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  setup        - Complete project setup (install tools, deps, migrate, css)"
	@echo "  install-tools - Install Go development tools (templ, goose, sqlc)"
	@echo "  generate     - Generate SQLC and templ files"
	@echo "  dev          - Start development server with Air (RECOMMENDED)"
	@echo "  run          - DEPRECATED: Use 'air' or 'make dev' instead"
	@echo "  build        - Build the application"
	@echo "  test         - Run Go tests"
	@echo "  lint         - Run linter"
	@echo "  migrate      - Run database migrations"
	@echo "  migrate-down - Rollback database migrations"
	@echo "  migrate-status - Show migration status"
	@echo "  sqlc-generate - Generate SQLC database code"
	@echo "  seed         - Seed database with sample data"
	@echo "  css          - Compile Tailwind CSS"
	@echo "  css-watch    - Watch and compile CSS changes"
	@echo "  images       - Optimize product images"
	@echo "  e2e          - Run Playwright E2E tests"
	@echo "  e2e-ui       - Run E2E tests in UI mode"
	@echo "  tidy         - Clean up Go dependencies"
	@echo "  clean        - Clean build artifacts and generated files"