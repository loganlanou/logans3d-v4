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

3. **ALWAYS test migrations before committing**:
   ```bash
   # Delete local database (safe pre-launch)
   rm -f ./data/database.db*

   # Test up
   goose -dir storage/migrations sqlite3 ./data/database.db up

   # Test down (reset runs all down migrations)
   goose -dir storage/migrations sqlite3 ./data/database.db reset

   # Test up again
   goose -dir storage/migrations sqlite3 ./data/database.db up
   ```

4. **SQLite-specific considerations**:
   - SQLite doesn't support `DROP COLUMN` - you must recreate the table
   - When recreating tables in Down migrations, use the exact schema from the previous migration state
   - Drop indexes BEFORE attempting table recreation
   - Recreate all indexes that existed before your migration

5. **Never commit untested migrations** - broken Down migrations will cause issues for other developers

### Migration Best Practices

- **Idempotent**: Migrations should be safe to run multiple times
- **Reversible**: Every Up migration must have a working Down migration
- **Atomic**: Each migration should represent a single logical change
- **Tested**: Always test both up and down before committing
- **No direct schema changes**: Never use `ALTER TABLE` or similar commands outside of migrations

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

## Admin Dashboard Styling

**CRITICAL: All admin dashboard pages MUST use the standard admin CSS classes.**

The admin dashboard has a predefined styling system that ensures consistency across all pages. When creating or modifying admin pages, you MUST use these classes instead of creating custom styles.

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

### Standard Page Structure

```templ
templ MyAdminPage(c echo.Context, data []MyData) {
    @layout.AdminBase(c, "Page Title") {
        @layout.AdminContainer() {
            <!-- Header -->
            <div class="flex justify-between items-center mb-6">
                <h1 class="admin-text-primary admin-text-2xl admin-font-bold">Page Title</h1>
                <a href="/admin/action" class="admin-btn admin-btn-primary">
                    + Add Item
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
                <div class="flex gap-4 flex-wrap">
                    <select
                        name="filter"
                        class="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                        onchange="window.location.href = updateQueryParam('filter', this.value)"
                    >
                        <option value="">All Items</option>
                        <option value="active">Active</option>
                    </select>
                </div>
            </div>

            <!-- Data Table -->
            <div class="admin-card">
                <div class="admin-card-header">
                    <h2 class="admin-card-title">Items ({ fmt.Sprintf("%d", len(data)) })</h2>
                </div>
                <div class="overflow-x-auto">
                    <table class="admin-table">
                        <thead>
                            <tr>
                                <th>Column 1</th>
                                <th>Column 2</th>
                                <th>Column 3</th>
                            </tr>
                        </thead>
                        <tbody>
                            for _, item := range data {
                                <tr onclick={ templ.ComponentScript{Call: fmt.Sprintf("window.location.href='/admin/items/%s'", item.ID)} } style="cursor: pointer;">
                                    <td>
                                        <span class="admin-text-primary admin-font-medium">{ item.Name }</span>
                                    </td>
                                    <td>{ item.Value }</td>
                                    <td>
                                        <span class="admin-text-muted">{ item.Status }</span>
                                    </td>
                                </tr>
                            }
                        </tbody>
                    </table>
                </div>
            </div>
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

### Detail Page Pattern

```templ
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
                <h1 class="admin-text-primary admin-text-2xl admin-font-bold">
                    { item.Name }
                </h1>
                <div class="text-sm admin-text-muted">
                    ID: { item.ID }
                </div>
            </div>

            <!-- Content Cards -->
            <div class="admin-card">
                <div class="admin-card-header">
                    <h2 class="admin-card-title">Information</h2>
                </div>
                <div class="p-6">
                    <p class="admin-text-primary">{ item.Description }</p>
                </div>
            </div>
        }
    }
}
```

### Common Mistakes to Avoid

1. **DON'T create custom card styling** - Use `admin-card`, `admin-card-header`, `admin-card-title`
2. **DON'T use raw Tailwind classes for tables** - Use `admin-table`
3. **DON'T use raw color classes for text** - Use `admin-text-primary`, `admin-text-muted`, etc.
4. **DON'T use HTMX for filters** - Use JavaScript `onchange` navigation
5. **DON'T create a "View" or "Actions" column** - Make the entire row clickable
6. **DON'T use inline styles except for `cursor: pointer`** on clickable rows

### Reference Implementation

See these files for correct implementation:
- Table listing: `views/admin/dashboard.templ` (Products table)
- Detail page: `views/admin/contacts.templ` (ContactDetail template)
- Filters: `views/admin/dashboard.templ` (Product filters)
