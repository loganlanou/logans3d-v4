# Logan's 3D Creations v4

Modern e-commerce website for 3D printing business, built with Go + Echo +
Templ + Alpine.js + Tailwind CSS.

## 🚀 Quick Start

### Prerequisites

- Go 1.25+
- Node.js 18+ (for CSS compilation and E2E tests)
- Air (hot reload tool)
- direnv (environment management)

### Installation

#### Prerequisites Setup

```bash
# 1. Install Go 1.25+ (if not already installed)
curl -fsSL https://go.dev/dl/go1.25.0.linux-amd64.tar.gz -o go1.25.0.linux-amd64.tar.gz
mkdir -p ~/go-install && tar -C ~/go-install -xzf go1.25.0.linux-amd64.tar.gz

# 2. Install direnv for environment management
# Ubuntu/Debian: sudo apt install direnv
# macOS: brew install direnv
# Then add to your shell config: eval "$(direnv hook bash)"

# 3. Install development tools
export GOROOT=~/go-install/go && export PATH=$GOROOT/bin:$PATH && export GOPATH=~/go
go install github.com/air-verse/air@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install github.com/pressly/goose/v3/cmd/goose@latest
go install github.com/a-h/templ/cmd/templ@latest
```

#### Project Setup

```bash
# Clone repository
git clone <repo-url>
cd logans3d-v4

# Setup environment variables
cp .envrc.example .envrc
# Edit .envrc with your actual values (see Environment Variables section below)
direnv allow  # Load environment variables

# Install dependencies and setup database
go mod tidy
make migrate  # Create and run database migrations

# Start development server
air  # Always use 'air' for development (never 'go run')
```

### Development

**⚠️ IMPORTANT: Always use Air for development, never `go run` or `make run`**

```bash
# Start development server (RECOMMENDED)
air

# Alternative (calls air with startup message)
make dev
```

**Why Air?**

- Automatically regenerates SQLC and Templ code via `pre_cmd`
- Hot reloads on file changes
- Prevents forgetting regeneration steps
- Configured in `.air.toml` with proper excludes and includes

### Available Commands

```bash
# Development
air              # Start development server (ALWAYS use this)
make dev         # Alternative: calls air
make build       # Build production binary
make generate    # Manual code generation (air does this automatically)

# Database
make migrate     # Run database migrations
make migrate-down # Rollback migrations  
make seed        # Seed with sample data

# Frontend
make css         # Compile Tailwind CSS
make css-watch   # Watch and recompile CSS

# Testing
make test        # Run Go tests
make e2e         # Run Playwright E2E tests
make lint        # Run linter

# Utilities
make clean       # Clean build artifacts
make help        # Show all commands
```

## 🏗️ Architecture

**Pattern**: Server-Side Rendered (SSR) with Progressive Enhancement

**Core Stack**:

- **Backend**: Go 1.25 + Echo v4.13
- **Database**: SQLite + SQLC + Goose (pure Go, no CGO)
- **Templates**: Templ (type-safe Go templates)
- **Frontend**: Alpine.js + Tailwind CSS v4
- **Development**: Air for hot reloading
- **Testing**: Playwright E2E + Go testing
- **Deployment**: Vercel serverless

## 📁 Project Structure

```text
logans3d-v4/
├── cmd/main.go              # Application entrypoint
├── service/                 # Business logic & HTTP handlers
│   ├── config.go           # Configuration management
│   ├── service.go          # Route registration & handlers
│   ├── handlers/           # Feature-specific handlers
│   └── middleware/         # Custom middleware
├── storage/                 # Data access layer
│   ├── storage.go          # Database connection
│   ├── sqlc.yaml          # SQLC configuration
│   ├── queries/           # SQL queries (SQLC input)
│   ├── migrations/        # Database schema migrations
│   └── db/               # Generated database code (SQLC output)
├── views/                   # Templ templates
│   ├── layout/            # Base layouts
│   ├── components/        # Reusable components
│   └── [feature]/        # Feature-specific templates
├── public/                  # Static assets
│   ├── css/              # Compiled CSS
│   ├── js/               # JavaScript files
│   └── images/           # Media assets
├── tests/                   # E2E test suites
└── scripts/                # Utility scripts
```

## 🗄️ Database

**Database**: SQLite with modernc.org/sqlite (pure Go, no CGO required)
**Migrations**: Goose (`storage/migrations/`)
**Queries**: SQLC for type-safe Go code generation (`storage/queries/`)

### Schema Overview

- **users**: Customer accounts (Google OAuth)
- **products & categories**: Product catalog with variants
- **orders & cart_items**: E-commerce functionality
- **quote_requests & quote_files**: Custom 3D printing quotes
- **events**: Maker faires, markets, shows
- **admin_users**: Admin access control

## 🎨 Frontend

**CSS Framework**: Tailwind CSS v4 with PostCSS
**JavaScript**: Alpine.js for progressive enhancement
**Templates**: Templ for type-safe server-side rendering

```bash
# CSS development
make css        # Compile once
make css-watch  # Watch for changes
```

## 🧪 Testing

**E2E Testing**: Playwright
**Unit Testing**: Go's built-in testing

```bash
npm test           # Run all E2E tests
npm run test:ui    # Run tests with UI
make test          # Run Go unit tests
```

## 🚀 Deployment

**Platform**: Vercel (serverless Go functions)
**Domain**: logans3dcreations.com
**SSL**: Automatic HTTPS with Vercel Edge Network
**Database**: SQLite with Litestream backup (production)

```bash
make deploy  # Deploy to production
```

## 📋 Environment Variables

Copy `.envrc.example` to `.envrc` and configure:

```bash
# Required for development
export GOROOT=~/go-install/go           # Path to Go installation
export PATH=$GOROOT/bin:$PATH           # Add Go to PATH
export GOPATH=~/go                      # Go workspace path

# Application settings
export ENVIRONMENT="development"
export PORT="8000"
export BASE_URL="http://localhost:8000"
export DB_PATH="./db/logans3d.db"

# Security (generate secure values for production)
export JWT_SECRET="development-jwt-secret-key-change-in-production"
export ADMIN_USERNAME="admin"
export ADMIN_PASSWORD="dev-admin-password"

# External services (optional for basic development)
export GOOGLE_CLIENT_ID=""              # Google OAuth (customer accounts)
export GOOGLE_CLIENT_SECRET=""
export STRIPE_PUBLISHABLE_KEY=""        # Stripe payments
export STRIPE_SECRET_KEY=""
export EMAIL_API_KEY=""                 # Email notifications
```

**Required for Development:**

- `GOROOT`, `PATH`, `GOPATH` - Go environment setup
- `ENVIRONMENT`, `PORT`, `BASE_URL` - Basic application config
- `DB_PATH` - Database location

**Optional for Basic Development:**

- OAuth, Stripe, Email - Can be added later for full functionality

## 📚 Documentation

- [Project Requirements](./REQUIREMENTS.md) - Detailed feature specifications
- [Implementation Plan](./PLAN.md) - Development roadmap and phases
- [Stack & Architecture](./STACK_AND_STRUCTURE.md) - Technical deep dive
- [Domain Transfer](./docs/DOMAIN_TRANSFER.md) - DNS migration guide

## 🔄 Development Workflow

1. **Start Development**: `air` (never `go run`)
2. **Make Changes**: Edit Go, Templ, or CSS files
3. **Auto-Reload**: Air automatically regenerates and reloads
4. **Database Changes**: Create migration files, run `make migrate`
5. **Frontend**: CSS compiles automatically, or use `make css-watch`
6. **Testing**: Run `make e2e` for full test suite

## 📈 Implementation Phases

- **Pre-Phase 1** ✅: Foundation, database, basic server
- **Phase 1** ⏳: Core pages and content management  
- **Phase 2** ⏳: Product catalog and shopping cart
- **Phase 3** ⏳: Checkout and custom quote system
- **Phase 4+** ⏳: Advanced features and optimizations

---

## Built with ❤️ for Logan's 3D Creations
