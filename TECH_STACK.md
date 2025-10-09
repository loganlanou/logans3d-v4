# Technology Stack Documentation

## Project Overview

**Logan's 3D Creations** - Modern e-commerce platform for a 3D printing business built with Go and server-side rendering.

## Backend Stack

### Core Framework & Language
- **Language**: Go 1.25
- **Web Framework**: [Echo v4.13.3](https://echo.labstack.com/)
  - High-performance, extensible web framework
  - Built-in middleware for logging, recovery, and CORS
  - Custom slog-based request logging middleware

### Database Layer
- **Database**: SQLite
  - **Driver**: [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) v1.38.2 (Pure Go implementation)
  - **CGO Driver**: [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) v1.14.32 (for tooling compatibility)
  - **Location**: `./data/database.db`
- **Query Builder**: [SQLC](https://sqlc.dev/)
  - Type-safe Go code generation from SQL
  - Configuration: `storage/sqlc.yaml`
  - Generated code: `storage/db/`
- **Migrations**: [Goose](https://github.com/pressly/goose)
  - Migration files: `storage/migrations/`
  - Commands available via Makefile

### Templating & Rendering
- **Template Engine**: [Templ](https://templ.guide/) v0.3.943
  - Type-safe Go HTML templates
  - Component-based architecture
  - Templates located in `views/` directory
  - Generated `_templ.go` files via `go generate`

### Authentication
- **Provider**: [Clerk](https://clerk.com/) v2.4.2
  - JWT-based authentication
  - Client and server-side integration
  - Session management
  - Implementation: `internal/auth/`, `internal/middleware/auth.go`

### Payment Processing
- **Provider**: [Stripe](https://stripe.com/) v80.2.1
  - Payment intents
  - Webhook handling
  - Implementation: `internal/stripe/`, `internal/handlers/payment.go`

### Shipping Integration
- **Provider**: ShipStation API
  - Rate calculation
  - Label generation
  - Package optimization
  - Implementation: `internal/shipping/`
  - Configuration: `config/shipping.json`

### Utilities & Libraries
- **UUID Generation**: [google/uuid](https://github.com/google/uuid) v1.6.0
- **Logging**: [lmittmann/tint](https://github.com/lmittmann/tint) v1.0.7 (Slog handler with color)
- **JWT**: [go-jose](https://github.com/go-jose/go-jose) v3.0.4 (via Clerk SDK)

## Frontend Stack

### Styling
- **CSS Framework**: [Tailwind CSS](https://tailwindcss.com/) v4.1.12
  - Utility-first CSS framework
  - PostCSS v8 for processing
  - Configuration: `postcss.config.js`
  - Input files: `public/css/public-input.css`, `public/css/admin-input.css`
  - Output files: `public/css/public-styles.css`, `public/css/admin-styles.css`

### JavaScript
- **Framework**: Vanilla JavaScript (no framework)
- **Key Modules**:
  - `cart.js` - Shopping cart logic
  - `cart-render.js` - Cart UI rendering
  - `clerk-init.js` - Client-side authentication
  - `custom-order.js` - Custom order forms
  - `shipping.js` - Shipping calculations
  - `scroll-control.js` - Scroll behavior management

### Assets
- **Location**: `public/`
  - `css/` - Stylesheets
  - `js/` - JavaScript files
  - `images/` - Static images
  - `uploads/` - User-uploaded content
- **Manifest**: `public/manifest.json` (PWA support)
- **SEO**: `robots.txt`, `sitemap.xml`

## Development Environment

### Environment Management
- **Tool**: [direnv](https://direnv.net/)
- **Configuration**: `.envrc`
- **Usage**: Automatically loads environment variables when entering the project directory
- **Key Variables**:
  - `PORT=8000` - Development server port
  - `DB_PATH=./data/database.db` - Database location
  - `ENVIRONMENT=development` - Runtime environment
  - API keys for Clerk, Stripe, ShipStation

### Hot Reload & Development Server
- **Tool**: [Air](https://github.com/cosmtrek/air)
- **Configuration**: `.air.toml`
- **Features**:
  - Auto-rebuild on file changes (.go, .templ, .css)
  - Pre-build commands:
    - Kill existing processes on port 8000
    - Run `go generate ./...` (generates Templ templates)
    - Run `npm run build:css` (compile Tailwind)
  - Excludes: test files, generated files, node_modules
- **Command**: `air` or `make dev`

### Build System
- **Build Tool**: GNU Make
- **Configuration**: `Makefile`
- **Key Targets**:
  - `make setup` - Complete project initialization
  - `make dev` - Start development server with Air
  - `make build` - Build production binary
  - `make migrate` - Run database migrations
  - `make sqlc-generate` - Generate database code
  - `make css` - Compile Tailwind CSS
  - `make seed` - Seed database with sample data
  - `make test` - Run tests

### Package Management
- **Go**: Go modules (`go.mod`, `go.sum`)
- **Node.js**: npm (`package.json`, `package-lock.json`)

## Project Structure

```
logans3d-v4/
├── cmd/                    # Application entry point
│   └── main.go            # Server initialization
├── internal/              # Private application code
│   ├── auth/             # Authentication service
│   ├── handlers/         # HTTP handlers
│   ├── middleware/       # Custom middleware
│   ├── shipping/         # Shipping logic & ShipStation
│   ├── stripe/           # Stripe integration
│   └── types/            # Custom type definitions
├── service/              # Business logic layer
│   ├── config.go         # Configuration management
│   └── service.go        # Service initialization
├── storage/              # Data layer
│   ├── db/               # Generated SQLC code
│   ├── migrations/       # Database migrations (Goose)
│   ├── queries/          # SQL query definitions (SQLC)
│   ├── scripts/          # Database utilities
│   └── sqlc.yaml         # SQLC configuration
├── views/                # Templ templates
│   ├── admin/            # Admin interface
│   ├── auth/             # Authentication pages
│   ├── components/       # Reusable components
│   ├── layout/           # Layout templates
│   ├── shop/             # E-commerce pages
│   └── ...               # Other page sections
├── public/               # Static assets
│   ├── css/              # Stylesheets (input & compiled)
│   ├── js/               # Client-side JavaScript
│   ├── images/           # Static images
│   └── uploads/          # User uploads
├── config/               # Application configuration files
├── data/                 # Runtime data (database, ignored by Git)
├── scripts/              # Utility scripts
├── .air.toml             # Air hot reload configuration
├── .envrc                # Environment variables (direnv)
├── Makefile              # Build automation
├── go.mod                # Go dependencies
├── package.json          # Node.js dependencies
└── postcss.config.js     # PostCSS configuration
```

## Architecture Patterns

### Server-Side Rendering (SSR)
- All HTML is rendered server-side using Templ templates
- No client-side framework (React, Vue, etc.)
- JavaScript used for progressive enhancement only

### Handler-Service-Repository Pattern
```
HTTP Request
    ↓
Echo Router
    ↓
Middleware (Auth, Logging, CORS)
    ↓
Handlers (internal/handlers/)
    ↓
Service Layer (service/)
    ↓
Repository (storage/db/ - SQLC generated)
    ↓
SQLite Database
```

### Code Generation Workflow
1. Write SQL queries in `storage/queries/*.sql`
2. Run `sqlc generate` to create type-safe Go code
3. Write Templ templates in `views/*.templ`
4. Run `go generate` to create Go rendering functions
5. Air automatically triggers these on file changes

## Testing

### End-to-End Testing
- **Framework**: Playwright (recently removed, but infrastructure exists)
- **Configuration**: MCP Playwright and Chrome DevTools integrations available
- **Coverage**: Previously tested admin interface, forms, orders, SEO

## Deployment

### Build Process
1. Pull latest code from Git
2. Run `go generate ./...` (regenerate templates)
3. Run `go build -o logans3d ./cmd` (compile binary)
4. Restart systemd service

### Environments
- **Staging**: logans3dcreations.digitaldrywood.com
  - Service: `logans3d-staging`
  - Deploy: `make deploy-staging`
- **Production**: www.logans3dcreations.com
  - Service: `logans3d`
  - Deploy: `make deploy-production`

### Server Configuration
- **Platform**: Linux (systemd services)
- **SSH**: apprunner@jarvis.digitaldrywood.com
- **Logs**: journalctl and file-based logs

## Third-Party Services

| Service | Purpose | Integration Points |
|---------|---------|-------------------|
| **Clerk** | Authentication & user management | `internal/auth/`, client-side JS |
| **Stripe** | Payment processing | `internal/stripe/`, `internal/handlers/payment.go` |
| **ShipStation** | Shipping & fulfillment | `internal/shipping/shipstation.go` |
| **SendGrid** | Transactional email (configured) | Environment variables only |

## Security Considerations

- **Authentication**: Clerk-managed JWT tokens
- **Environment Variables**: Managed via direnv, not committed to Git
- **Database**: SQLite with file-based permissions
- **CORS**: Configured via Echo middleware
- **File Uploads**: Max size 100MB, stored in `public/uploads/`
- **Input Validation**: Type-safe SQLC queries prevent SQL injection

## Performance Optimizations

- **SQLite**: Lightweight, zero-config database
- **Pure Go**: modernc.org/sqlite eliminates CGO overhead in production
- **Static Assets**: Served directly by Echo
- **CSS**: Tailwind compiled to optimized CSS files
- **Hot Reload**: Air watches only relevant file types

## Developer Experience

### First-Time Setup
```bash
# Install direnv (if not already installed)
# Then:
make setup           # Install tools, dependencies, migrate DB
direnv allow         # Enable environment variables
air                  # Start development server
```

### Daily Development
```bash
air                  # Start dev server (auto-reloads on changes)
make css-watch       # Watch CSS changes (if needed separately)
```

### Common Tasks
```bash
make migrate         # Apply database migrations
make seed            # Seed database with sample data
make sqlc-generate   # Regenerate database code
go generate ./...    # Regenerate Templ templates
```

## Future Considerations

- **Database Scaling**: Consider PostgreSQL for high-traffic production
- **Asset Pipeline**: Consider Vite or esbuild for JavaScript bundling
- **Testing**: Reinstate Playwright E2E tests
- **Caching**: Add Redis for session/cart storage
- **CDN**: Serve static assets via CDN in production
- **Monitoring**: Add observability (Sentry, DataDog, etc.)
