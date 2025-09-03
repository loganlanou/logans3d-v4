# Logan's 3D Creations Technology Stack & Architecture Deep Dive

_Last updated: 2025-09-03_

## 🏗️ Architecture Overview

Logan's 3D Creations will be built with a **Go backend** serving **server-side rendered HTML** using **Templ templates**, enhanced with **Alpine.js** for interactivity, styled with **Tailwind CSS**, and backed by **SQLite** with **SQLC** for type-safe database access.

**Architecture Pattern**: Server-Side Rendered (SSR) with Progressive Enhancement
**Deployment**: Single binary deployment (pure Go, no CGO required)
**Inspiration**: Based on the proven CreswoodCorners technology stack

---

## 🔧 Core Technology Stack

### Backend Technologies

#### **Go 1.25** - Primary Language
- **Web Framework**: [Echo v4.13+](https://echo.labstack.com/) - High performance HTTP router and middleware framework
- **Template Engine**: [Templ](https://templ.guide/) - Type-safe Go HTML templates with component architecture
- **Authentication**: 
  - **OAuth2**: Google OAuth integration with [golang.org/x/oauth2](https://pkg.go.dev/golang.org/x/oauth2)
  - **JWT**: [golang-jwt/jwt/v5](https://github.com/golang-jwt/jwt) for session management
- **Logging**: [lmittmann/tint](https://github.com/lmittmann/tint) - Structured logging with colored output
- **UUID Generation**: [google/uuid](https://github.com/google/uuid)

#### **Key Go Dependencies** (Planned):
```go
// Core web framework and routing
github.com/labstack/echo/v4 v4.13+

// HTML templating with type safety
github.com/a-h/templ v0.3.943+

// Authentication & security  
github.com/golang-jwt/jwt/v5 v5.3.0+
golang.org/x/oauth2 v0.30.0+

// Database & migrations (pure Go, no CGO)
modernc.org/sqlite v1.28.0+
github.com/pressly/goose/v3 v3.25.0+

// Utilities
github.com/google/uuid v1.6.0+
github.com/lmittmann/tint v1.0.7+

// Payments
github.com/stripe/stripe-go/v75
```

### Frontend Technologies

#### **Templ Templates** - Server-Side Rendering
- Type-safe Go HTML templates with component architecture
- Server-side rendering with SEO-friendly markup
- Component-based template organization in `views/` directory:
  - `layout/` - Base templates and page layouts
  - `components/` - Reusable UI components 
  - `ui/components/` - UI library components (buttons, inputs, cards, etc.)
  - Feature-specific directories (`shop/`, `custom/`, `portfolio/`, etc.)

#### **Alpine.js 3.x** - Frontend Interactivity
- Lightweight JavaScript framework for reactive UI components
- Progressive enhancement approach - JavaScript enhances server-rendered HTML
- Used for:
  - Shopping cart interactions (add/remove, quantity updates)
  - Product image galleries and zoom
  - Form validation and dynamic fields
  - Mobile navigation and dropdowns
  - Custom quote form interactions

#### **Tailwind CSS v4+** - Styling Framework  
- **PostCSS** for CSS processing with `@tailwindcss/postcss`
- Utility-first CSS framework for rapid UI development
- **Configuration**: `postcss.config.js` processes `public/css/input.css` → `public/css/styles.css`
- **Custom Classes**: Component-specific styling with utility classes
- **Mobile-first responsive design** for phone and desktop optimization

#### **Google Identity Services** - OAuth Integration
- Client-side Google OAuth flow for customer accounts
- Loaded via CDN: `https://accounts.google.com/gsi/client`

### Database & Data Layer

#### **SQLite 3** - Primary Database
- **Driver**: `modernc.org/sqlite` - Pure Go SQLite3 driver (no CGO required)
- **Location**: `./db/logans3d.db` (configurable via `DB_PATH` env var)
- **Benefits**: 
  - Zero-configuration database
  - ACID compliance
  - Perfect for single-server deployments
  - File-based backup and replication with Litestream
  - No CGO dependency - easier cross-compilation and deployment

#### **SQLC** - Type-Safe SQL Code Generation  
- **Configuration**: `storage/sqlc.yaml`
- **Features**:
  - Generates type-safe Go code from SQL queries
  - Compile-time query validation
  - Auto-generated database models and query methods
- **Structure**:
  - `storage/queries/` - Raw SQL query definitions
  - `storage/migrations/` - Database schema migrations  
  - `storage/db/` - Generated Go code (models, queries, interfaces)

#### **Goose** - Database Migrations
- **Migration Directory**: `storage/migrations/`
- **Planned Schema**:
  - Products (name, description, price, variants, images, categories)
  - Categories (name, slug, description)
  - Orders (customer info, line items, status, payment data)
  - Custom Quotes (contact info, files, specifications, status)
  - Users (OAuth profiles, order history, preferences)
  - Events (name, date, location, description)
- **Usage**: `make migrate` / `make migrate-down`

#### **Litestream** - Database Backup & Replication
- **Purpose**: Continuous backup of SQLite database to cloud storage
- **Destinations**: S3, GCS, Azure, or similar
- **Benefits**: Point-in-time recovery, disaster recovery, zero-downtime backups

---

## 🛠️ Development Workflow & Build Tools

### **Make** - Build Automation
The `Makefile` will provide comprehensive development commands:

```makefile
# Development
make dev          # Run development server with hot reloading
make generate     # Generate SQLC and Templ files  
make build        # Build production binary (pure Go)

# Database
make migrate      # Run database migrations up
make migrate-down # Rollback migrations
make seed         # Seed database with sample products

# Testing & Quality
make test         # Run Go tests  
make lint         # Run golangci-lint
make e2e          # Run Playwright E2E tests

# CSS & Assets
make css          # Compile Tailwind CSS
make images       # Optimize product images

# Deployment
make deploy       # Deploy to production
make logs         # View production logs

# Utilities  
make clean        # Clean build artifacts
make help         # Show available commands
```

### **Air** - Hot Reloading for Development
- **Configuration**: `.air.toml`
- **Features**:
  - Watches `.go`, `.templ`, `.html`, `.css` files
  - Auto-runs `go generate ./...` before compilation
  - Excludes test files and generated files (`_test.go`, `_templ.go`)
  - Builds to `./tmp/logans3d` for development
- **Usage**: `air` (auto-detects `.air.toml`)

### **Code Generation Pipeline**
```bash
go generate ./...
```
Runs:
1. **SQLC**: Generates type-safe database code from SQL files
2. **Templ**: Compiles `.templ` templates to Go functions

### **CSS Build Process** 
```bash
# PostCSS processes Tailwind CSS
postcss public/css/input.css -o public/css/styles.css
```
- **Input**: `public/css/input.css` (Tailwind directives + custom CSS)
- **Output**: `public/css/styles.css` (compiled utility classes)
- **Configuration**: `postcss.config.js` with `@tailwindcss/postcss` plugin

---

## 🧪 Testing & Quality Assurance

### **Playwright** - End-to-End Testing
- **Configuration**: `playwright.config.ts` 
- **Test Directory**: `tests/`
- **Critical User Journeys**:
  - Product browsing and search
  - Add to cart and checkout flow
  - Custom quote submission
  - Account creation and login
  - Admin product management
  - Mobile responsive behavior

**Test Commands:**
```bash
npm test           # Run all tests
npm run test:ui    # Run tests in UI mode  
npm run test:headed # Run tests in headed mode
npm run test:report # Show test results
```

### **Go Testing** - Unit Tests
- **Command**: `make test` 
- **Coverage**: Go's built-in testing framework
- **Focus Areas**: Database queries, business logic, API endpoints

### **Linting & Code Quality**
- **Tool**: [golangci-lint](https://golangci-lint.run/)
- **Usage**: `make lint`
- **Configuration**: `.golangci.yml` for custom rules

---

## 🚀 Deployment & Infrastructure

### **Production Environment**
- **Hosting**: [Vercel](https://vercel.com) - Free plan with automatic deployments
- **Runtime**: Vercel's Go runtime for serverless functions
- **Domain**: logans3dcreations.com (to be transferred from Square to DNSimple)
- **SSL**: Automatic HTTPS with Vercel's Edge Network

### **Deployment Process**
**Vercel Deployment**:
- **Git Integration**: Automatic deployments on push to main branch
- **Build Command**: `go build ./cmd/main.go` 
- **Environment Variables**: Configured in Vercel dashboard
- **Database**: SQLite file stored in Vercel's serverless file system
- **Static Assets**: Served from Vercel's CDN

**Manual Deploy**:
```bash
vercel --prod  # Deploy to production
vercel         # Deploy preview branch
```

### **Vercel Configuration**
- **vercel.json**: Project configuration file
- **Go Runtime**: Serverless functions for API endpoints
- **Static Files**: Automatic optimization and CDN delivery
- **Environment**: Separate staging and production environments

### **Backup & Monitoring**
- **Database**: Continuous backup with Litestream
- **Application Logs**: Structured logging with log aggregation
- **Health Monitoring**: Basic health check endpoints
- **Analytics**: GA4 + ecommerce event tracking

---

## 📁 Project Structure Deep Dive

```
logans3d-v4/
├── 📄 Configuration Files
│   ├── go.mod, go.sum           # Go module dependencies
│   ├── package.json             # Node.js dependencies (Tailwind, Playwright)
│   ├── Makefile                 # Build automation and commands
│   ├── .air.toml               # Hot reloading configuration  
│   ├── postcss.config.js       # CSS processing configuration
│   ├── playwright.config.ts    # E2E testing configuration
│   ├── .envrc                  # Environment variables (not committed)
│   └── .gitignore              # Version control exclusions
│
├── 🏗️ Application Code
│   ├── cmd/                    # Application entrypoints
│   │   └── main.go            # Primary application binary
│   │
│   ├── service/               # Business logic layer
│   │   ├── config.go         # Environment configuration  
│   │   ├── service.go        # Route registration and middleware
│   │   ├── handlers/         # HTTP request handlers
│   │   │   ├── shop.go       # Product catalog handlers
│   │   │   ├── cart.go       # Shopping cart handlers
│   │   │   ├── checkout.go   # Stripe integration
│   │   │   ├── custom.go     # Custom quote handlers
│   │   │   ├── admin.go      # Admin panel handlers
│   │   │   └── auth.go       # Authentication handlers
│   │   └── middleware/       # Custom middleware
│   │
│   ├── storage/              # Data access layer
│   │   ├── storage.go        # Database connection and setup
│   │   ├── config.go         # Database configuration
│   │   ├── sqlc.yaml        # SQLC code generation config
│   │   ├── db/              # Generated database code (SQLC)
│   │   ├── queries/         # SQL query definitions  
│   │   ├── migrations/      # Database schema migrations
│   │   └── seed/            # Database seeding scripts
│   │
│   └── views/               # Template layer (Templ)
│       ├── layout/          # Base page layouts
│       ├── components/      # Shared template components
│       ├── ui/components/   # UI library components
│       ├── home/           # Homepage templates  
│       ├── shop/           # Product catalog templates
│       ├── custom/         # Custom quote templates
│       ├── portfolio/      # Portfolio/gallery templates
│       ├── admin/          # Admin panel templates
│       └── auth/           # Authentication templates
│
├── 🎨 Frontend Assets
│   └── public/
│       ├── css/
│       │   ├── input.css    # Tailwind source file
│       │   └── styles.css   # Compiled CSS output
│       ├── js/
│       │   ├── app.js       # Main application JavaScript
│       │   ├── cart.js      # Shopping cart functionality
│       │   ├── gallery.js   # Image gallery interactions
│       │   └── admin.js     # Admin panel functionality
│       ├── images/          # Product images and media
│       │   ├── products/    # Product photography
│       │   ├── portfolio/   # Portfolio/gallery images
│       │   └── assets/      # Logos, icons, etc.
│       └── uploads/         # User-uploaded files (STL, etc.)
│
├── 🧪 Testing & Scripts  
│   ├── tests/              # Playwright E2E tests
│   │   ├── shop.spec.ts    # Shopping flow tests
│   │   ├── custom.spec.ts  # Custom quote tests
│   │   └── admin.spec.ts   # Admin panel tests
│   ├── test-results/       # Test execution results
│   └── scripts/           # Utility scripts
│       ├── seed-db/       # Database seeding
│       └── image-process/ # Image optimization
│
└── 📋 Documentation & Meta
    ├── README.md           # Project setup and overview
    ├── REQUIREMENTS.md     # Detailed project requirements
    ├── STACK_AND_STRUCTURE.md  # This document
    └── docs/              # Additional documentation
        ├── DEPLOYMENT.md   # Deployment guide
        ├── API.md         # API documentation
        └── CONTRIBUTING.md # Development guide
```

---

## 🔐 Environment & Configuration Management

### **direnv** - Environment Variable Management
- **File**: `.envrc` (not committed to version control)
- **Installation**: `brew install direnv` or `sudo apt install direnv`
- **Setup**: `direnv allow` in project directory

#### **Required Environment Variables**:
```bash
# Application
export ENVIRONMENT="development"
export PORT="8000"
export BASE_URL="http://localhost:8000"

# Database
export DB_PATH="./db/logans3d.db"

# OAuth Configuration (for customer accounts)
export GOOGLE_CLIENT_ID="your-google-client-id"
export GOOGLE_CLIENT_SECRET="your-google-client-secret"  
export GOOGLE_REDIRECT_URL="http://localhost:8000/auth/google/callback"

# JWT Security
export JWT_SECRET="your-super-secret-jwt-key"

# Stripe Payment Processing
export STRIPE_PUBLISHABLE_KEY="pk_test_..."
export STRIPE_SECRET_KEY="sk_test_..."
export STRIPE_WEBHOOK_SECRET="whsec_..."

# Email (Transactional)
export EMAIL_FROM="noreply@logans3dcreations.com"
export EMAIL_PROVIDER="sendgrid"  # or mailgun, ses, etc.
export EMAIL_API_KEY="your-email-api-key"

# File Uploads
export UPLOAD_MAX_SIZE="104857600"  # 100MB
export UPLOAD_DIR="./public/uploads"

# Admin Access (development only)
export ADMIN_USERNAME="admin"
export ADMIN_PASSWORD="secure-dev-password"

# Backup (production)
export LITESTREAM_ACCESS_KEY_ID="your-s3-key"
export LITESTREAM_SECRET_ACCESS_KEY="your-s3-secret"
export LITESTREAM_BUCKET="logans3d-backups"
```

---

## 🔄 Data Flow & Request Lifecycle

### **Typical Request Flow**:
1. **HTTP Request** → Echo router (`service/service.go`)
2. **Middleware** → Authentication, logging, CORS, rate limiting
3. **Handler** → Business logic in `service/handlers/` package  
4. **Database** → SQLC-generated type-safe queries with pure Go SQLite
5. **Template** → Templ renders server-side HTML with data
6. **Response** → HTML + CSS + minimal JavaScript sent to browser
7. **Enhancement** → Alpine.js adds client-side interactivity

### **E-commerce Flow**:
1. **Product Browsing** → Server renders product grid with filters
2. **Add to Cart** → Alpine.js updates cart state + AJAX to server
3. **Checkout** → Server renders checkout form with Stripe Elements
4. **Payment** → Stripe processes payment, webhooks update order status
5. **Confirmation** → Server renders success page + sends confirmation email

### **Custom Quote Flow**:
1. **Quote Form** → Server renders form with file upload capabilities
2. **File Upload** → Multi-part form submission with validation
3. **Quote Processing** → Server stores files, sends admin notification
4. **Admin Review** → Admin panel for quote management
5. **Customer Follow-up** → Email communication with quote details

---

## 🎯 Key Design Decisions & Trade-offs

### **Why This Stack?**

1. **Go + Echo**: High performance, excellent concurrency, simple deployment
2. **Templ**: Type-safe templates with Go's compile-time guarantees  
3. **SQLite + SQLC**: Zero-config database with type-safe queries, perfect for small e-commerce
4. **Alpine.js**: Progressive enhancement without heavy framework overhead
5. **Server-Side Rendering**: SEO-friendly for product pages, fast initial loads
6. **Pure Go (no CGO)**: Simplified cross-compilation, deployment, and maintenance
7. **modernc.org/sqlite**: Pure Go SQLite driver eliminates CGO complexity

### **Trade-offs Made**:

**✅ Benefits**:
- Fast development velocity with type safety across the stack
- Simple deployment and maintenance (single binary + database file)
- Excellent performance and low hosting costs
- SEO-friendly product pages for organic discovery
- Strong security with minimal attack surface
- Mobile-first responsive design
- Easy cross-compilation for any target platform
- No CGO dependencies simplify builds and deployments

**⚠️ Considerations**:
- SQLite limits to single-server deployment (fine for small e-commerce)
- Smaller Go ecosystem for some e-commerce features vs Node.js/PHP
- Manual payment webhook handling vs all-in-one solutions
- Pure Go SQLite driver may have slightly different performance characteristics vs CGO version

---

## 🚀 Business-Specific Features

### **Product Management**
- **Product Variants**: Size, material, color options
- **Inventory Tracking**: Stock levels, low stock alerts
- **Category Management**: Hierarchical product categorization
- **Image Management**: Multiple product images, optimization
- **Lead Time Tracking**: Production time estimates per product

### **Custom Quote System**
- **File Upload**: STL, OBJ, STEP, 3MF file support up to 100MB
- **Quote Request Form**: Project details, materials, finish requirements
- **Admin Workflow**: Quote review, pricing, approval process
- **Customer Communication**: Email notifications and updates

### **E-commerce Features**
- **Shopping Cart**: Persistent cart, quantity updates
- **Checkout Flow**: Guest and logged-in user checkout
- **Payment Processing**: Stripe integration with Apple Pay/Google Pay
- **Order Management**: Status tracking, fulfillment workflow
- **Local Pickup**: Option for event/local pickup

### **Content Management**
- **Event Listings**: Upcoming shows, maker faires, markets
- **Portfolio Gallery**: Showcase custom work with categories
- **Educational Content**: 3D printing process explanation
- **Mobile Admin**: Edit products and content from phone

---

## 📈 Future Considerations & Scaling Path

### **Phase 1 Priorities** (MVP):
- Basic product catalog with cart/checkout
- Custom quote form with file upload
- Content pages (about, events, contact)
- Mobile-responsive admin panel

### **Phase 2 Enhancements**:
- Customer accounts with order history
- Advanced product filtering and search
- Inventory management and alerts
- Email marketing integration

### **Phase 3+ Advanced Features**:
- Customer reviews and testimonials
- Advanced analytics and reporting
- Loyalty program or repeat customer discounts
- API for potential mobile app

### **Scaling Options When Needed**:
1. **CDN**: Static asset delivery for images
2. **Database**: Migrate to PostgreSQL if needed (SQLC supports both)
3. **Caching**: Redis for session storage, product cache
4. **Microservices**: Extract admin or custom quote system
5. **Load Balancing**: Multiple app instances behind load balancer

---

## 📚 Key Resources & Documentation

- **Go Echo Framework**: https://echo.labstack.com/
- **Templ Templates**: https://templ.guide/  
- **SQLC Documentation**: https://docs.sqlc.dev/
- **modernc.org/sqlite**: https://pkg.go.dev/modernc.org/sqlite
- **Alpine.js Guide**: https://alpinejs.dev/
- **Tailwind CSS**: https://tailwindcss.com/
- **Playwright Testing**: https://playwright.dev/
- **Stripe Integration**: https://stripe.com/docs/payments/checkout
- **Litestream Backup**: https://litestream.io/

---

_This document serves as the definitive guide to Logan's 3D Creations' technology stack and should be updated as the architecture evolves._