# AGENTS.md

Go HTTP server built with the standard library (`net/http`, Go 1.25+) as part of the boot.dev backend course. See `README.md` for the project overview and setup instructions.

## Architecture

- `main.go` — server entry point and route registration
- `internal/handler/` — HTTP handlers and middleware; handlers share state via the `apiConfig` type in `api_config.go`
- `internal/auth/` — password hashing (argon2id), JWT, refresh tokens
- `internal/database/` — **generated sqlc code; never hand-edit**. Change `sql/queries/*.sql` instead, then run `sqlc generate`
- `sql/schema/` — goose migrations; apply with `./goose.sh up` (requires `.env` with `DB_URL`)
- `requests/*.http` — REST Client request examples, organized one file per endpoint group

## Build and Test

- Build: `go build ./...`
- Test: `go test ./...`
- Vet: `go vet ./...`
- Run: `go run .` with `DB_URL` set in `.env`

Run build and tests after any change before reporting done.

## Conventions

- Keep `requests/*.http` in sync whenever endpoints change — the user relies on these files to test endpoints
- Handler patterns: early returns, `returnWithJSON`/`returnWithError` helpers in `respond.go`, request/response structs at package level
- Prefer modern stdlib idioms (Go 1.25): `atomic.Int32`, `fmt.Appendf`, etc. — the editor's linter flags older patterns
- sqlc regenerates `internal/database/` — edit the SQL queries, never the generated files

## Communication

The user is learning Go. When asked to explain something ("explain this", "why", "how does this work"):

- Explain step by step in plain language with minimal jargon
- Use concrete analogies when helpful
- Verify the user's understanding is correct before moving on

When asked to evaluate or review code ("evaluate", "take a new look"):

- Do one complete pass with a full checklist (naming, consistency, dead code, idiomatic patterns) instead of pointing out issues one at a time
- Run `go build ./...` and `go test ./...` as part of the evaluation
