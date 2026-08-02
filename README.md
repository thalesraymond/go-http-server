# Go HTTP Server

A RESTful API server built in Go as part of the [boot.dev](https://boot.dev) backend course. This project demonstrates core concepts in HTTP server development, database integration, and API design using Go's standard library and PostgreSQL.

## About

This project is part of the boot.dev curriculum for learning Go and backend development. It implements a simple "Chirp" (tweet-like) API with user authentication, showcasing practical patterns for building production-ready HTTP services in Go.

## Features

- **HTTP Server** — Built using Go's `net/http` package with Go 1.22+ route patterns
- **PostgreSQL Integration** — Database layer using `database/sql` with `lib/pq` driver
- **Type-safe SQL** — Generated queries via [sqlc](https://sqlc.dev/)
- **User Management** — Registration and authentication endpoints
- **Chirp API** — Create, read, update, and delete chirps (tweets)
- **Input Validation** — Chirp content validation
- **Metrics** — Request counting and health check endpoints
- **Static File Serving** — Serves HTML files from the project root

## Project Structure

```
├── main.go                 # Server entry point & route registration
├── internal/
│   ├── database/           # Generated sqlc database layer
│   │   ├── db.go
│   │   ├── models.go
│   │   ├── users.sql.go
│   │   └── chirps.sql.go
│   └── handler/            # HTTP handlers & middleware
│       ├── api_config.go   # Shared config (DB, metrics)
│       ├── user.go         # User endpoints
│       ├── chirp.go        # Chirp endpoints
│       ├── healthz.go      # Health check
│       ├── respond.go      # Response helpers
│       └── validate_chirp.go
├── sql/
│   ├── schema/             # Database migrations
│   └── queries/            # sqlc query definitions
└── requests/               # HTTP request examples (REST Client)
```

## API Endpoints

| Method | Endpoint              | Description            |
| ------ | --------------------- | ---------------------- |
| GET    | `/api/healthz`        | Health check           |
| POST   | `/api/validate_chirp` | Validate chirp content |
| GET    | `/admin/metrics`      | Get request metrics    |
| POST   | `/admin/reset`        | Reset metrics          |
| GET    | `/app/*`              | Static file server     |

Additional user and chirp endpoints are registered via `RegisterRoutes` methods in their respective handlers.

## Prerequisites

- Go 1.22+
- PostgreSQL
- [sqlc](https://sqlc.dev/) (for regenerating database code)

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
   # Apply schema from sql/schema/ directory
   psql $DB_URL -f sql/schema/001_create_users.up.sql
   psql $DB_URL -f sql/schema/002_create_chirps.up.sql
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

### Testing endpoints

The `requests/` directory contains `.http` files compatible with VS Code's REST Client extension or similar tools.

## License

MIT License — see [LICENSE](LICENSE) file for details.
