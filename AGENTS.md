# Repository Guidelines

## Project Structure & Module Organization
- `cmd/main.go` boots the HTTP server and wires core services.
- `service/` holds business logic, Echo route registration, and middleware.
- `storage/` manages SQLite access: `queries/` (SQLC input), `db/` (generated code), and `migrations/`.
- `views/` contains Templ components and layouts; match feature folders with `service/handlers`.
- `public/` serves compiled Tailwind assets; `tests/` houses Playwright suites; `scripts/` collects CLI helpers.

## Build, Test, and Development Commands
- `air` — primary dev loop; regenerates Templ/SQLC output before hot reloads.
- `make dev` — shorthand wrapper around `air` with project-specific logging.
- `make generate` — manually regenerates Templ components and SQLC bindings.
- `make build` — compiles the production Go binary with static assets embedded.
- `make migrate` / `make migrate-down` — runs Goose migrations forward or back.
- `make test` / `make e2e` / `make lint` — Go unit tests, Playwright E2E run, `golangci-lint` sweep.

## Coding Style & Naming Conventions
- Go code follows `gofmt` defaults (tabs for indentation, camelCase locals, exported PascalCase). Run `gofmt` or rely on `air`'s pre-commands.
- Generated code lives under `storage/db/`; never edit it manually.
- Templ components in `views/` use PascalCase filenames (e.g., `ProductCard.templ`); colocate style helpers in the same directory.
- Tailwind utility classes stay in markup; avoid custom CSS unless shared across pages (`public/css/input.css`).
- JavaScript sprinkled via Alpine.js components; prefer descriptive `x-data` keys (e.g., `cartDrawer`).

## Testing Guidelines
- Write Go tests in `_test.go` files adjacent to the code; ensure table-driven cases cover success and failure paths.
- Target minimum 80% coverage on new packages; validate locally with `go test ./...`.
- For UI flows, add Playwright specs under `tests/e2e/`; name specs after user journeys (`checkout.spec.ts`).
- Before pushing, run `make test` and `make e2e` when UI changes or auth flows shift.

## Commit & Pull Request Guidelines
- Follow the existing imperative tense style (`Prevent Clerk handshake redirect loops`).
- Keep commits focused; include config or schema snapshots when they change.
- Pull requests must summarize intent, list key changes, and link GitHub issues or task IDs.
- Attach screenshots or terminal output for UI and auth updates; flag any migration steps in the description.
- Request review from platform maintainers and wait for CI (lint, unit, E2E) to succeed before merge.
