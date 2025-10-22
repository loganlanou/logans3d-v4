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

## Environment Management

This project uses **direnv** to manage environment variables:

- Environment variables are configured in `.envrc`
- After making changes to `.envrc`, run `direnv allow` to activate them
- The environment is automatically loaded when entering the directory (if direnv is installed)

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
