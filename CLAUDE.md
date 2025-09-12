# Claude AI Assistant Notes

## Database Configuration

**Official database location**: `./data/database.db`

- The SQLite database file is stored in the `./data/` directory
- Environment variable `DB_PATH` in `.envrc` points to `./data/database.db`
- The `./data/` directory is ignored by Git (configured in `.gitignore`)
- This resolves the naming collision that previously existed between the
  top-level `./db/` directory (actual database files) and the `./storage/db/`
  directory (generated Go database code)

## Environment Management

This project uses **direnv** to manage environment variables:

- Environment variables are configured in `.envrc`
- After making changes to `.envrc`, run `direnv allow` to activate them
- The environment is automatically loaded when entering the directory (if direnv is installed)

To make environment changes:
1. Update `.envrc` 
2. Run `direnv allow`
