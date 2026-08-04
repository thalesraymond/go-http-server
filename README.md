# Go HTTP Server

[![CI](https://github.com/thalesraymond/go-http-server/actions/workflows/ci.yml/badge.svg)](https://github.com/thalesraymond/go-http-server/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/thalesraymond/go-http-server)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A RESTful API server built in Go as part of the [boot.dev](https://boot.dev) backend course. This project demonstrates core concepts in HTTP server development, database integration, and API design using Go's standard library and PostgreSQL.

## About

This project is part of the boot.dev curriculum for learning Go and backend development. It implements a simple "Chirp" (tweet-like) API with user authentication, showcasing practical patterns for building production-ready HTTP services in Go.

## Features

- **HTTP Server** — Built using Go's `net/http` package with Go 1.22+ route patterns
- **PostgreSQL Integration** — Database layer using `database/sql` with `lib/pq` driver
- **Type-safe SQL** — Generated queries via [sqlc](https://sqlc.dev/)
- **Database Migrations** — Managed with [goose](https://github.com/pressly/goose)
- **User Management** — Registration and authentication endpoints
- **Authentication** — Password hashing (argon2id), JWT access tokens, refresh tokens with revoke
- **Chirp API** — Create, read, update, and delete chirps (tweets)
- **Webhooks** — Polka webhook endpoint for external event handling
- **Input Validation** — Chirp content validation
- **Metrics** — Request counting and health check endpoints
- **Middleware** — Auth middleware for protected routes
- **Static File Serving** — Serves HTML files from the project root

## Project Structure

```
├── main.go                       # Server entry point & route registration
├── internal/
│   ├── auth/                     # Auth primitives
│   │   ├── hashing.go            # Argon2id password hashing
│   │   ├── jwt.go                # JWT access tokens
│   │   └── refresh_token.go      # Opaque refresh tokens
│   ├── database/                 # Generated sqlc database layer
│   │   ├── db.go
│   │   ├── models.go
│   │   ├── users.sql.go
│   │   ├── chirps.sql.go
│   │   └── refresh_tokens.sql.go
│   └── handler/                  # HTTP handlers & middleware
│       ├── admin.go              # Metrics & reset endpoints
│       ├── logger.go             # Logger interface & default implementation
│       ├── user.go               # User endpoints
│       ├── login.go              # Login / refresh / revoke endpoints
│       ├── chirp.go              # Chirp endpoints
│       ├── polka.go              # Polka webhook handler
│       ├── middleware.go         # Auth handshake (RequireAuth / RequirePolkaAuth)
│       ├── healthz.go            # Health check
│       ├── respond.go            # Response helpers
│       └── validate_chirp.go     # Chirp content validation
├── sql/
│   ├── schema/                   # Goose migration files
│   └── queries/                  # sqlc query definitions
├── requests/                     # REST Client request examples
└── goose.sh                      # Wrapper to run goose with .env
```

## API Endpoints

| Method | Endpoint              | Description                   |
| ------ | --------------------- | ----------------------------- |
| GET    | `/api/healthz`        | Health check                  |
| GET    | `/admin/metrics`      | Get request metrics           |
| POST   | `/admin/reset`        | Reset metrics and data        |
| GET    | `/app/*`              | Static file server            |
| POST   | `/api/users`          | Register a new user           |
| PUT    | `/api/users`          | Update current user (auth)    |
| POST   | `/api/login`          | Log in, returns JWT + refresh |
| POST   | `/api/refresh`        | Refresh access token          |
| POST   | `/api/revoke`         | Revoke a refresh token        |
| POST   | `/api/chirps`         | Create a chirp (auth)         |
| GET    | `/api/chirps`         | List all chirps               |
| GET    | `/api/chirps/{id}`    | Get a chirp by ID             |
| DELETE | `/api/chirps/{id}`    | Delete a chirp (auth)         |
| POST   | `/api/polka/webhooks` | Polka webhook (auth)          |

## Prerequisites

- Go 1.22+
- PostgreSQL
- [sqlc](https://sqlc.dev/) (for regenerating database code)
- [goose](https://github.com/pressly/goose) (for database migrations)

## Setup

1. **Clone the repository:**

   ```bash
   git clone https://github.com/thalesraymond/go-http-server.git
   cd go-http-server
   ```

2. **Create `.env` file:**

   ```bash
   DB_URL=postgres://user:password@localhost:5432/dbname?sslmode=disable
   ```

3. **Run database migrations:**

   ```bash
   ./goose.sh up
   ```

4. **Start the server:**
   ```bash
   go run main.go
   ```

The server starts on `:8080`.

## Development

### Regenerate SQL code

```bash
sqlc generate
```

### Run tests

Tests run through [gotestsum](https://github.com/gotesttools/gotestsum) (pinned as a Go tool in `go.mod`) with the race detector and coverage:

```bash
go tool gotestsum --format standard-verbose -- -v -race -cover ./...
```

### Testing endpoints

The `requests/` directory contains `.http` files compatible with VS Code's REST Client extension or similar tools.

## License

MIT License — see [LICENSE](LICENSE) file for details.
