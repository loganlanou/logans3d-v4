SHELL := /bin/bash

.PHONY: install-tools
install-tools:
	go install github.com/a-h/templ/cmd/templ@latest
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

.PHONY: setup-hooks
setup-hooks:
	@echo "⚙️  Setting up git hooks..."
	@if [ ! -d .git ]; then \
		echo "❌ Not a git repository"; \
		exit 1; \
	fi
	@git config core.hooksPath .githooks
	@chmod +x .githooks/*
	@echo "✅ Git hooks configured! Pre-commit hook will check formatting and vet."

.PHONY: templ-update
templ-update:
	go install github.com/a-h/templ/cmd/templ@latest
	go get -u github.com/a-h/templ@latest

.PHONY: templ-fmt
templ-fmt:
	templ fmt .

.PHONY: reset
reset:
	# first, ask if they want to reset the database, it's highly destructive
	@read -p "Are you sure you want to reset the database? This will delete all data and cannot be undone. (y/N): " confirm && [ "$$confirm" = "y" ] || (echo "Database reset cancelled." && exit 1)
	# remove the database
	rm ./data/database.db
	# run the migrations
	make migrate
	# seed the database
	make seed

.PHONY: generate
generate:
	go generate ./...

.PHONY: dev
dev:
	@if [ -f tmp/air-combined.log ]; then \
		mv tmp/air-combined.log tmp/air-combined-$$(date +%Y%m%d-%H%M%S).log; \
	fi
	@ls -t tmp/air-combined-*.log 2>/dev/null | tail -n +6 | xargs rm -f 2>/dev/null || true
	@air 2>&1 | tee tmp/air-combined.log

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
	@echo "Checking integration tests compile..."
	go test -tags integration -run '^$$' ./...

.PHONY: test-coverage
test-coverage:
	go test -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: test-integration
test-integration:
	go test -tags=integration ./... -v

.PHONY: test-verbose
test-verbose:
	go test -v ./...

.PHONY: ci
ci: lint test build
	@echo "✅ All CI checks passed!"

.PHONY: lint
lint:
	golangci-lint run --max-issues-per-linter=0 || echo "Linter not installed. Run make install-linter"

.PHONY: install-linter
install-linter:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v2.1.6

.PHONY: migrate
migrate:
	mkdir -p data && goose -dir storage/migrations sqlite3 ./data/database.db up

.PHONY: migrate-down
migrate-down:
	goose -dir storage/migrations sqlite3 ./db/logans3d.db down

.PHONY: migrate-status
migrate-status:
	goose -dir storage/migrations sqlite3 ./db/logans3d.db status

.PHONY: test-migrations
test-migrations:
	@echo "🧪 Testing database migrations..."
	@cd scripts/test-migrations && go run main.go

.PHONY: sqlc-generate
sqlc-generate:
	sqlc generate -f storage/sqlc.yaml

.PHONY: seed
seed:
	go run scripts/seed-products/main.go -db ./data/database.db

.PHONY: admins
admins:
	go run scripts/make-lanou-admins/main.go -db ./data/database.db

.PHONY: css
css:
	npx postcss public/css/input.css -o public/css/styles.css

.PHONY: css-watch
css-watch:
	npx postcss public/css/input.css -o public/css/styles.css --watch

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
setup: install-tools tidy setup-hooks
	npm install
	mkdir -p db
	make migrate
	make css
	@echo "Setup complete! Run 'air' to start development server (with auto-regeneration)"

# Deployment targets
.PHONY: ssh
ssh:
	ssh -A apprunner@jarvis.digitaldrywood.com

.PHONY: deploy-production
deploy-production:
	@echo "🚀 Deploying to production (www.logans3dcreations.com)..."
	@read -p "⚠️  Are you sure you want to deploy to PRODUCTION? (y/N): " confirm && [ "$$confirm" = "y" ] || (echo "Deployment cancelled." && exit 1)
	ssh -A apprunner@jarvis.digitaldrywood.com "cd /home/apprunner/sites/logans3d && git pull && /usr/local/go/bin/go generate ./... && /usr/local/go/bin/go build -o logans3d ./cmd && sudo systemctl restart logans3d"
	@echo "✅ Production deployment complete!"

.PHONY: deploy
deploy: deploy-production

.PHONY: log
log:
	ssh -A apprunner@jarvis.digitaldrywood.com "sudo journalctl -u logans3d -f"

.PHONY: log-web
log-web:
	ssh -A apprunner@jarvis.digitaldrywood.com "sudo tail -f /var/log/logans3d/logans3d.log"

.PHONY: env-view
env-view:
	@echo "📋 Production environment variables:"
	@ssh -A apprunner@jarvis.digitaldrywood.com "sudo cat /etc/logans3d/environment"

.PHONY: env-set
env-set:
	@if [ -z "$(filter-out $@,$(MAKECMDGOALS))" ]; then \
		echo "❌ Usage: make env-set KEY=VALUE"; \
		echo "Example: make env-set EMAIL_FROM=prints@logans3dcreations.com"; \
		exit 1; \
	fi
	@KEY_VALUE='$(filter-out $@,$(MAKECMDGOALS))'; \
	echo "🔧 Setting environment variable on production server..."; \
	ssh -A apprunner@jarvis.digitaldrywood.com "echo $$KEY_VALUE | sudo tee -a /etc/logans3d/environment > /dev/null && sudo systemctl restart logans3d"; \
	echo "✅ Variable set: $$KEY_VALUE"; \
	echo "✅ Service restarted"

%:
	@:

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  setup        - Complete project setup (install tools, deps, migrate, css, hooks)"
	@echo "  install-tools - Install Go development tools (templ, goose, sqlc)"
	@echo "  setup-hooks  - Configure git pre-commit hooks (gofmt, go vet)"
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
	@echo ""
	@echo "Deployment:"
	@echo "  ssh              - SSH to the deployment server"
	@echo "  deploy           - Deploy to production (same as deploy-production)"
	@echo "  deploy-production - Deploy to production (www.logans3dcreations.com)"
	@echo "  log  - View production logs (journalctl)"
	@echo "  log-web - View production web logs"
	@echo ""
	@echo "Environment Management:"
	@echo "  env-view         - View production environment variables"
	@echo "  env-set KEY=VALUE - Set production environment variable (restarts service)"