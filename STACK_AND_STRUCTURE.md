# Project Stack & Structure Deep Dive

_Last updated: 2025-09-03_

## Environment Management: direnv + .envrc

- **direnv** will be used for environment variable management instead of a plain `.env` file.
- Project root will include a `.envrc` file, which loads environment variables for local development.
- Example `.envrc`:
  ```sh
  export GOOGLE_CLIENT_ID="..."
  export GOOGLE_CLIENT_SECRET="..."
  export GOOGLE_REDIRECT_URL="http://localhost:8000/auth/google/callback"
  export JWT_SECRET="your-super-secret-jwt-key"
  export ENVIRONMENT="development"
  export DB_PATH="./db/creswoodcorners.db"
  export ADMIN_USERNAME="admin"
  export ADMIN_PASSWORD="fireworks2025"
  # Add any other secrets or config here
  ```
- **How to use:**
  1. Install [direnv](https://direnv.net/) (`brew install direnv` or `sudo apt install direnv`)
  2. Hook direnv into your shell (see `direnv hook` docs)
  3. Add `.envrc` to your project root and allow it with `direnv allow`
  4. Environment variables will be automatically loaded/unloaded when you `cd` into/out of the project directory.

## Example Directory Structure

```
project-root/
├── .envrc                # direnv config for environment variables
├── Makefile              # Build/dev commands
├── README.md             # Project overview and setup
├── go.mod, go.sum        # Go module files
├── cmd/
│   ├── main.go           # Application entrypoint
│   └── generate.go       # Code generation (templ, sqlc)
├── service/
│   ├── config.go         # Loads config from env
│   ├── service.go        # Business logic, route registration
│   └── ...               # Handlers, middleware, etc.
├── storage/
│   ├── storage.go        # DB connection, migrations
│   ├── config.go         # DB config struct
│   ├── db/               # SQLC-generated code
│   ├── migrations/       # SQL migration files
│   └── queries/          # SQLC query definitions
├── views/
│   ├── layout/           # Base templates
│   ├── home/             # Homepage templates
│   ├── products/         # Product-related templates
│   └── ...               # Other page/component templates
├── public/
│   ├── css/              # Compiled CSS (Tailwind)
│   ├── js/               # JS (Alpine.js, htmx, etc.)
│   ├── images/           # Static images
│   └── ...
├── scripts/              # Utility scripts (e.g., import-csv)
├── tests/                # Playwright E2E tests
└── ...
```

## Notes for Future Planning
- All secrets/config should be managed via `.envrc` and **never** committed to version control.
- Document any new required environment variables in this file and in the README.
- Use `direnv reload` after editing `.envrc`.
- For CI/CD, use environment variable injection (not `.envrc`).

---

This document should be updated as the stack or structure evolves.
