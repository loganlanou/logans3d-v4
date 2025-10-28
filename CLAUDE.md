# Claude AI Assistant Notes

## Development Workflow

### IMPORTANT: `make dev` is Always Running

**The development server is ALWAYS running** - started by the user with `make dev`.

**AI assistants / agents should NEVER run `make dev` or any development server commands.**

The user manages the development server, and it's always available. AI assistants should:

- **NEVER** run `make dev`, `air`, or any server startup commands
- **ALWAYS** assume the dev server is running
- **Check logs** in `./tmp/` directory when troubleshooting (logs are always available)
- **Make code changes** and let Air automatically detect and rebuild

### What `make dev` Does (For Reference)

`make dev` runs Air, which provides automatic hot-reloading and handles all code generation:

- **Automatically runs code generation** via `go generate ./...` on every file change
  - Generates templ files (`.templ` → `_templ.go`)
  - Generates sqlc database code
- **Automatically compiles CSS** via `npm run build:css` (Tailwind CSS)
- **Hot reloads** the Go application on file changes
- **Cleans up port 8000** before starting (kills any existing process)
- **Watches** for changes in `.go`, `.templ`, and `.css` files

### AI Assistants: Never Run These Commands

AI assistants should **NEVER** run these commands, as the user manages the dev server:

- `make dev` (user only - always running)
- `air` (user only - always running)
- `go generate ./...` (Air runs this automatically)
- `templ generate` (part of `go generate`)
- `sqlc generate` (part of `go generate`)
- `npm run build:css` or `npx postcss` (Air runs this automatically)
- `make generate` (Air handles this)
- `make css` or `make css-watch` (Air handles this)

### Development Logs (Always Available)

All development logs are **always available** in the `./tmp/` directory:

- `./tmp/air-combined.log` - Current combined output (stdout + stderr)
- `./tmp/build-errors.log` - Build errors from Air
- `./tmp/air-combined-YYYYMMDD-HHMMSS.log` - Archived historical logs

**AI assistants can read these logs at any time** to troubleshoot build issues or check application output.

**Note**: `make dev` keeps only the 5 most recent archived logs and automatically rotates them.

## Database Configuration

**Official database location**: `./data/database.db`

- The SQLite database file is stored in the `./data/` directory
- Environment variable `DB_PATH` in `.envrc` points to `./data/database.db`
- The `./data/` directory is ignored by Git (configured in `.gitignore`)
- This resolves the naming collision that previously existed between the
  top-level `./db/` directory (actual database files) and the `./storage/db/`
  directory (generated Go database code)

### Database Migrations

**CRITICAL: Migrations run automatically on server startup.**

- Migrations are embedded in the binary using `//go:embed` in `storage/storage.go`
- They run automatically after database connection in `storage.New()`
- Goose tracks which migrations have been applied and only runs new ones
- Migration files are located in `./storage/migrations/`

**IMPORTANT: Database Schema Changes MUST Be Made Via Migrations**

AI assistants must NEVER make database schema changes directly. All schema changes MUST be made through migration files.

### Creating and Testing Migrations

When creating a new migration:

1. **Create the migration file**:

   ```bash
   goose -dir storage/migrations create migration_name sql
   ```

2. **Write the Up and Down migrations**:
   - The `Up` section applies the change
   - The `Down` section MUST correctly reverse the change
   - For SQLite, remember that `DROP COLUMN` is not supported - you must recreate the table

3. **CRITICAL: ONLY TEST THE MIGRATION YOU JUST CREATED**:

   ```bash
   # WRONG - NEVER DO THIS - DESTROYS ALL DATA
   rm -f ./data/database.db*
   goose -dir storage/migrations sqlite3 ./data/database.db reset

   # CORRECT - Test only your new migration
   # First, ensure you're on the version BEFORE your migration
   goose -dir storage/migrations sqlite3 ./data/database.db status

   # Test UP for your specific migration
   goose -dir storage/migrations sqlite3 ./data/database.db up-by-one

   # Test DOWN for your specific migration
   goose -dir storage/migrations sqlite3 ./data/database.db down

   # Test UP again
   goose -dir storage/migrations sqlite3 ./data/database.db up-by-one
   ```

4. **NEVER use `goose reset` or delete the database unless**:
   - You have explicitly confirmed there is NO production data
   - You have a verified backup
   - The user has explicitly requested it
   - **When in doubt, ASK THE USER FIRST**

5. **SQLite-specific considerations**:
   - SQLite doesn't support `DROP COLUMN` - you must recreate the table
   - When recreating tables in Down migrations, use the exact schema from the previous migration state
   - Drop indexes BEFORE attempting table recreation
   - Recreate all indexes that existed before your migration

6. **Never commit untested migrations** - broken Down migrations will cause issues for other developers

### Migration Best Practices

- **Idempotent**: Migrations should be safe to run multiple times
- **Reversible**: Every Up migration must have a working Down migration
- **Atomic**: Each migration should represent a single logical change
- **Tested**: Always test both up and down before committing
- **No direct schema changes**: Never use `ALTER TABLE` or similar commands outside of migrations

### SQLite Date/Time Handling with Go

**CRITICAL: Understanding how go-sqlite3 stores time.Time is essential to avoid query bugs.**

#### How go-sqlite3 Stores time.Time

When Go's `time.Time` is inserted into SQLite using the `mattn/go-sqlite3` driver:

- **Storage Format**: TEXT as ISO8601 string WITH TIMEZONE SUFFIX
- **Example**: `2025-10-27 21:23:41 +0000 UTC` (NOT standard ISO8601)
- **Problem**: SQLite's `DATE()`, `datetime()`, and `strftime()` functions CANNOT parse the timezone suffix

#### The Recurring Bug Pattern

**WRONG - This will return NULL/empty dates:**
```sql
SELECT DATE(abandoned_at) as date FROM abandoned_carts
-- Returns empty because DATE() can't parse "2025-10-27 21:23:41 +0000 UTC"
```

**CORRECT - Strip timezone first:**
```sql
SELECT DATE(substr(abandoned_at, 1, 10)) as date FROM abandoned_carts
-- Returns "2025-10-27" correctly

SELECT strftime('%Y-%m-%d', substr(abandoned_at, 1, 19)) as date FROM abandoned_carts
-- Returns "2025-10-27" correctly (handles time portion too)
```

#### Best Practices for SQLite Date/Time Queries

1. **Always strip timezone info when using SQLite date functions:**
   - Use `substr(column_name, 1, 10)` for date-only: `YYYY-MM-DD`
   - Use `substr(column_name, 1, 19)` for datetime: `YYYY-MM-DD HH:MM:SS`

2. **Common query patterns:**
   ```sql
   -- Extract date for grouping
   GROUP BY DATE(substr(created_at, 1, 10))

   -- Extract hour for time-of-day analysis
   strftime('%H', substr(created_at, 1, 19))

   -- Format for display
   strftime('%Y-%m-%d %H:%M', substr(created_at, 1, 19))

   -- Date range queries (these work fine without substr)
   WHERE created_at >= datetime('now', '-7 days')
   WHERE created_at BETWEEN '2025-01-01' AND '2025-12-31'
   ```

3. **Schema design:**
   - Use `DATETIME` or `TIMESTAMP` as column type (for documentation, SQLite treats as TEXT)
   - Always store times in UTC (Go: `time.Now().UTC()`)
   - Convert to local timezone only in application/display layer

4. **Testing queries:**
   ```bash
   # Always test date extraction queries in sqlite3 CLI first
   sqlite3 ./data/database.db "SELECT substr(created_at, 1, 10), created_at FROM table LIMIT 5;"
   ```

#### Why This Matters

**Impact**: Every time you use `DATE()`, `strftime()`, or time-based grouping without `substr()`, you get NULL/empty results, breaking:
- Analytics charts (empty labels)
- Time-based aggregations (wrong groupings)
- Date filtering (no results)

**Remember**: Go stores with timezone → SQLite functions need clean ISO8601 → Use `substr()` to strip timezone

## Configuration Management

**CRITICAL: ALL application configuration MUST be stored in the database.**

- **NEVER use local JSON/YAML config files** for application settings
- Configuration files like `config/shipping.json` should NOT be used
- All configuration must be stored in database tables and loaded at runtime
- Use migrations to add new configuration tables/columns as needed

### Why Database-Only Configuration

1. **Single Source of Truth**: Database is the only source of configuration
2. **Admin UI**: Configuration can be managed through admin interface
3. **No File Sync Issues**: Eliminates configuration drift between environments
4. **Version Control**: Database migrations track configuration schema changes
5. **Runtime Updates**: Configuration can be updated without redeploying code

### Configuration Loading Pattern

```go
// CORRECT: Load from database
config, err := queries.GetShippingConfig(ctx)

// WRONG: Never do this
config := loadConfigFromJSONFile("config/shipping.json")
```

## Environment Management

This project uses **direnv** to manage environment variables:

- Environment variables are configured in `.envrc`
- After making changes to `.envrc`, run `direnv allow` to activate them
- The environment is automatically loaded when entering the directory (if direnv is installed)
- **Environment variables are for secrets and environment-specific settings only**, NOT for application configuration

To make environment changes:

1. Update `.envrc`
2. Run `direnv allow`

## Image Path Architecture

**IMPORTANT: Product images follow a strict separation between database storage and view rendering.**

### Database Storage (product_images table)

- **ONLY store the filename** (e.g., `pachycephalosaurus.jpg`)
- **NEVER store paths** like `/public/images/products/` in the database
- This keeps the database portable and allows path changes without database migrations

### View Layer (Service Handlers)

- The view layer constructs the full public path when rendering
- Pattern: `imageURL = "/public/images/products/" + filename`
- This happens in:
  - `service/service.go` - `handleShop()`, `handleShopCategory()`, `handlePremium()`, `handleProduct()`
  - `internal/handlers/admin.go` - Admin product listings

### Verification

- Run `sqlite3 ./data/database.db "SELECT COUNT(*) FROM product_images WHERE image_url LIKE '%/%';"`
- Should return `0` - no paths in the database
- The script `scripts/fix-image-urls-correct/main.go` can clean up any incorrect path storage

### File System Location

- Product images are stored at: `./public/images/products/`
- Served at URL: `http://localhost:8000/public/images/products/filename.jpg`

## UI Component Library

**CRITICAL: ALL admin dashboards and pages MUST use TemplUI components.**

This project uses [TemplUI](https://templui.io/docs/components) (v0.98.0) - a library of 45+ beautiful, accessible UI components for Go templ.

### Installing TemplUI (Already Installed)

TemplUI is already installed in this project. To add components to your pages:

```go
import (
    "github.com/templui/templui/internal/components/card"
    "github.com/templui/templui/internal/components/button"
    "github.com/templui/templui/internal/components/table"
    "github.com/templui/templui/internal/components/dialog"
)
```

### Core TemplUI Components (Use These!)

**Cards:**

```templ
@card.Card() {
    @card.Header() {
        @card.Title() {
            Order Summary
        }
        @card.Description() {
            View order details below
        }
    }
    @card.Content() {
        <p>Your content here</p>
    }
    @card.Footer(card.FooterProps{
        Class: "flex justify-between",
    }) {
        @button.Button() {
            Action
        }
    }
}
```

**Buttons:**

```templ
// Primary button
@button.Button(button.Props{
    Variant: button.VariantPrimary,
}) {
    Save Changes
}

// Secondary button
@button.Button(button.Props{
    Variant: button.VariantSecondary,
    Size:    button.SizeSm,
}) {
    Cancel
}

// Destructive button
@button.Button(button.Props{
    Variant: button.VariantDestructive,
}) {
    Delete
}
```

**Tables:**

```templ
@table.Table() {
    @table.Header() {
        @table.Row() {
            @table.Head() { Name }
            @table.Head() { Email }
            @table.Head() { Status }
        }
    }
    @table.Body() {
        for _, item := range items {
            @table.Row() {
                @table.Cell() { item.Name }
                @table.Cell() { item.Email }
                @table.Cell() { item.Status }
            }
        }
    }
}
```

**Dialogs/Modals:**

```templ
@dialog.Content(dialog.ContentProps{
    ID: "myModal",
    Class: "max-w-2xl",
}) {
    @dialog.Header() {
        @dialog.Title() {
            Confirm Action
        }
    }
    <div class="p-6">
        <p>Are you sure?</p>
    </div>
    @dialog.Footer() {
        @dialog.Close() {
            @button.Button(button.Props{
                Variant: button.VariantOutline,
            }) {
                Cancel
            }
        }
        @button.Button(button.Props{
            Variant: button.VariantPrimary,
        }) {
            Confirm
        }
    }
}
```

### Available TemplUI Components

**Form & Input:** Input, Label, Textarea, Checkbox, Radio, Switch, Select Box, Tags Input, Date Picker, Time Picker, Input OTP, Form

**Display & Layout:** Card, Badge, Avatar, Separator, Skeleton, Aspect Ratio

**Navigation:** Button, Breadcrumb, Pagination, Tabs, Sidebar, Dropdown

**Feedback:** Alert, Toast, Tooltip, Progress, Rating, Code, Copy Button

**Advanced:** Dialog, Popover, Sheet, Accordion, Collapsible, Carousel, Calendar, Charts, Icon, Table

### Migration from Custom CSS to TemplUI

**WRONG (Old Pattern):**

```templ
<div class="admin-card">
    <div class="admin-card-header">
        <h2 class="admin-card-title">Title</h2>
    </div>
    <div class="p-6">Content</div>
</div>

<button class="admin-btn admin-btn-primary">Click Me</button>

<table class="admin-table">
    <thead><tr><th>Header</th></tr></thead>
    <tbody><tr><td>Data</td></tr></tbody>
</table>
```

**CORRECT (TemplUI Pattern):**

```templ
@card.Card() {
    @card.Header() {
        @card.Title() { Title }
    }
    @card.Content() {
        Content
    }
}

@button.Button(button.Props{
    Variant: button.VariantPrimary,
}) {
    Click Me
}

@table.Table() {
    @table.Header() {
        @table.Row() {
            @table.Head() { Header }
        }
    }
    @table.Body() {
        @table.Row() {
            @table.Cell() { Data }
        }
    }
}
```

### Component Priority Rules

1. **ALWAYS use TemplUI components** when available (Card, Button, Table, Dialog, etc.)
2. **ONLY use custom CSS classes** for styling not covered by TemplUI (like `admin-text-primary` for text colors)
3. **NEVER create raw HTML equivalents** of TemplUI components (no `<div class="admin-card">`, use `@card.Card()` instead)

### Reference Implementation

See `views/admin/orders.templ` for correct TemplUI usage:

- Lines 10-11: Component imports
- Lines 678-712: Dialog component with proper syntax
- Lines 705-709: Button component usage

## Legacy Admin CSS Classes (Being Deprecated)

**NOTE: The custom admin CSS classes below are DEPRECATED. Use TemplUI components instead.**

These classes are only listed for reference when maintaining existing code. All new code MUST use TemplUI components.

### Required CSS Classes

#### Card Components

- `admin-card` - Card container with proper background, border, and shadow
- `admin-card-header` - Card header section with border-bottom
- `admin-card-title` - Card title styling (use `<h2>` element)

#### Tables

- `admin-table` - Table with proper borders, spacing, and hover effects
- Tables must be wrapped in `<div class="overflow-x-auto">` for responsive scrolling
- Table headers use `<th>` without additional classes (styling is automatic)
- Table rows automatically get hover states

#### Typography

- `admin-text-primary` - Primary text color (dark)
- `admin-text-muted` - Muted/secondary text color (gray)
- `admin-text-disabled` - Disabled text color (light gray)
- `admin-text-warning` - Warning text color (yellow/orange)
- `admin-font-bold` - Bold font weight
- `admin-font-medium` - Medium font weight
- `admin-text-2xl` - 2xl text size

#### Buttons

- `admin-btn` - Base button class (required for all buttons)
- `admin-btn-primary` - Primary action button (blue)
- `admin-btn-secondary` - Secondary action button (gray)
- `admin-btn-danger` - Destructive action button (red)
- `admin-btn-warning` - Warning action button (orange)
- `admin-btn-sm` - Small button variant

### Standard Admin Page Structure (Using TemplUI)

```templ
package admin

import (
    "github.com/labstack/echo/v4"
    "github.com/loganlanou/logans3d-v4/views/layout"
    "github.com/templui/templui/internal/components/card"
    "github.com/templui/templui/internal/components/button"
    "github.com/templui/templui/internal/components/table"
)

templ MyAdminPage(c echo.Context, data []MyData) {
    @layout.AdminBase(c, "Page Title") {
        @layout.AdminContainer() {
            <!-- Header -->
            <div class="flex justify-between items-center mb-6">
                <h1 class="text-2xl font-bold text-gray-900">Page Title</h1>
                <a href="/admin/action">
                    @button.Button(button.Props{
                        Variant: button.VariantPrimary,
                    }) {
                        + Add Item
                    }
                </a>
            </div>

            <!-- Filters (if needed) -->
            <div class="mb-6 space-y-4">
                <input
                    type="text"
                    placeholder="Search..."
                    class="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    onkeyup="debounceSearch(this.value)"
                />
            </div>

            <!-- Data Table with TemplUI Card -->
            @card.Card() {
                @card.Header() {
                    @card.Title() {
                        Items ({ fmt.Sprintf("%d", len(data)) })
                    }
                }
                @card.Content() {
                    @table.Table() {
                        @table.Header() {
                            @table.Row() {
                                @table.Head() { Column 1 }
                                @table.Head() { Column 2 }
                                @table.Head() { Column 3 }
                            }
                        }
                        @table.Body() {
                            for _, item := range data {
                                @table.Row() {
                                    @table.Cell() {
                                        <span class="font-medium text-gray-900">{ item.Name }</span>
                                    }
                                    @table.Cell() { item.Value }
                                    @table.Cell() {
                                        <span class="text-gray-600">{ item.Status }</span>
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
```

### Filter/Navigation Pattern

Filters MUST use JavaScript navigation (NOT HTMX):

```javascript
<script>
    function updateQueryParam(key, value) {
        const url = new URL(window.location);
        if (value) {
            url.searchParams.set(key, value);
        } else {
            url.searchParams.delete(key);
        }
        return url.toString();
    }

    let searchTimeout;
    function debounceSearch(value) {
        clearTimeout(searchTimeout);
        searchTimeout = setTimeout(() => {
            window.location.href = updateQueryParam('search', value);
        }, 500);
    }
</script>

<select onchange="window.location.href = updateQueryParam('status', this.value)">
```

### Clickable Table Rows

Table rows should be clickable for navigation to detail pages:

```templ
<tr onclick={ templ.ComponentScript{Call: fmt.Sprintf("window.location.href='/admin/items/%s'", item.ID)} } style="cursor: pointer;">
    <td>Content</td>
</tr>
```

For links within clickable rows (like email links), stop propagation:

```templ
<a href="mailto:email@example.com" onclick="event.stopPropagation()">
    email@example.com
</a>
```

### Detail Page Pattern (Using TemplUI)

```templ
package admin

import (
    "github.com/labstack/echo/v4"
    "github.com/loganlanou/logans3d-v4/views/layout"
    "github.com/templui/templui/internal/components/card"
    "github.com/templui/templui/internal/components/button"
)

templ ItemDetail(c echo.Context, item MyItem) {
    @layout.AdminBase(c, "Item Details") {
        @layout.AdminContainer() {
            <!-- Back Button -->
            <div class="mb-6">
                <a href="/admin/items" class="text-blue-600 hover:text-blue-800">
                    ← Back to Items
                </a>
            </div>

            <!-- Header -->
            <div class="flex justify-between items-start mb-6">
                <h1 class="text-2xl font-bold text-gray-900">
                    { item.Name }
                </h1>
                <div class="text-sm text-gray-600">
                    ID: { item.ID }
                </div>
            </div>

            <!-- Content Cards using TemplUI -->
            @card.Card() {
                @card.Header() {
                    @card.Title() {
                        Information
                    }
                }
                @card.Content() {
                    <p class="text-gray-900">{ item.Description }</p>
                }
                @card.Footer(card.FooterProps{
                    Class: "flex justify-end gap-2",
                }) {
                    @button.Button(button.Props{
                        Variant: button.VariantSecondary,
                    }) {
                        Edit
                    }
                    @button.Button(button.Props{
                        Variant: button.VariantDestructive,
                    }) {
                        Delete
                    }
                }
            }
        }
    }
}
```

### Common Mistakes to Avoid

1. **DON'T use `<div class="admin-card">`** - Use `@card.Card()` instead
2. **DON'T use `<button class="admin-btn">`** - Use `@button.Button()` instead
3. **DON'T use `<table class="admin-table">`** - Use `@table.Table()` instead
4. **DON'T forget to import components** - Always import from `github.com/templui/templui/internal/components/...`
5. **DON'T use HTMX for filters** - Use JavaScript `onchange` navigation
6. **DON'T create a "View" or "Actions" column** - Make the entire row clickable

### Reference Implementations

**CORRECT TemplUI Usage:**

- `views/admin/orders.templ` lines 10-11: Component imports
- `views/admin/orders.templ` lines 678-712: Dialog component with Button
- `views/admin/orders.templ` lines 705-709: Proper Button component usage

**INCORRECT (Legacy Code - Needs Migration):**

- Most of `views/admin/dashboard.templ` - Uses `<div class="admin-card">` instead of `@card.Card()`
- Most of `views/admin/contacts.templ` - Uses raw HTML instead of TemplUI components

**Migration Priority:**

When editing any admin page, convert raw HTML components to TemplUI components.
