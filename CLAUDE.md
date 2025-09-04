# Claude AI Assistant Notes

## Database Configuration

**Official database location**: `./data/database.db`

- The SQLite database file is stored in the `./data/` directory
- Environment variable `DB_PATH` in `.envrc` points to `./data/database.db`
- The `./data/` directory is ignored by Git (configured in `.gitignore`)
- This resolves the naming collision that previously existed between the
  top-level `./db/` directory (actual database files) and the `./storage/db/`
  directory (generated Go database code)

## Development Commands

To run the application with proper environment variables:

```bash
source .envrc
```
