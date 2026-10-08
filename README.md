# Hospital Middleware

A Go-based API middleware service built with [Gin Web Framework](https://github.com/gin-gonic/gin), featuring PostgreSQL database integration and an Nginx reverse proxy with SSL/TLS (HTTPS) support.

---

## Table of Contents

- [Features](#features)
- [Architecture & Services](#architecture--services)
- [Project Structure](#project-structure)
- [Prerequisites](#prerequisites)
- [Quick Start for Developers](#quick-start-for-developers)
- [Getting Started](#getting-started)
  - [1. Configure Environment](#1-configure-environment)
  - [2. Generate SSL Certificates](#2-generate-ssl-certificates)
  - [3. Run with Docker Compose](#3-run-with-docker-compose)
  - [4. Run Locally (Go Development)](#4-run-locally-go-development)
- [API Endpoints & Testing](#api-endpoints--testing)
- [Database Migrations](#database-migrations)
- [Database Seeder (Mock Data)](#database-seeder-mock-data)
- [Testing (TDD)](#testing-tdd)
- [Environment Variables](#environment-variables)
- [Troubleshooting](#troubleshooting)

---

## Features

- **Gin HTTP Server**: Fast and lightweight REST API framework in Go.
- **Nginx Reverse Proxy**: Production-ready gateway with HTTP-to-HTTPS automatic redirect, HTTP/2, keepalive, and WebSocket support.
- **SSL/TLS (HTTPS)**: Preconfigured Nginx reverse proxy with SAN-enabled SSL certificates supporting `localhost` and `127.0.0.1`.
- **PostgreSQL 18**: Database layer with connection pooling, health probes, and automatic schema migrations.
- **Dockerized**: Multi-stage minimal non-root Alpine container with automated healthcheck orchestration.
- **Database Migrations**: Embedded migrations using `golang-migrate` and Go `embed.FS`.

---

## Architecture & Services

When running with Docker Compose, three services are orchestrated in the `hospital-network` bridge network:

```
[Client / Browser / curl]
         │
         ├─────────────────────────────────────────┐
         ▼ (Port 80 / 443)                         ▼ (Port 8080 - Optional Debug)
┌─────────────────────────────────┐       ┌─────────────────────────────────┐
│              Nginx              │       │          Go API Service         │
│              (nginx)            │──────>│     (hospital-middleware-api)   │
│  - Port 80 (HTTP -> 301 HTTPS)  │       │  - Port 8080                    │
│  - Port 443 (HTTPS / SSL)       │       └────────────────┬────────────────┘
└─────────────────────────────────┘                        │ (Port 5432)
                                                           ▼
                                          ┌─────────────────────────────────┐
                                          │            PostgreSQL           │
                                          │            (postgres)           │
                                          │  - Port 5432                    │
                                          └─────────────────────────────────┘
```

| Service | Container Name | Internal Port | Exposed Host Port | Description |
| :--- | :--- | :--- | :--- | :--- |
| **`nginx`** | `nginx` | `80`, `443` | `80`, `443` | Reverse proxy & SSL termination |
| **`hospital-middleware-api`** | `hospital-middleware-api` | `8080` | `8080` | Go Gin REST API application |
| **`postgres`** | `postgres` | `5432` | `5432` | PostgreSQL 18.4 database |

---

## Project Structure

```text
hospital-middleware/
├── cmd/
│   ├── api/
│   │   ├── main.go               # Application entrypoint & dependency injection
│   │   └── main_test.go          # API route integration tests
│   └── seed/
│       └── main.go               # Standalone database seeder CLI
├── internal/
│   ├── config/
│   │   ├── config.go             # Configuration & environment variable loader
│   │   └── config_test.go        # Config unit tests
│   ├── database/
│   │   ├── migrate.go            # Embedded migration executor (golang-migrate)
│   │   ├── migrate_test.go       # Migration runner unit tests
│   │   ├── postgres.go           # PostgreSQL connection pool & management
│   │   ├── postgres_test.go      # Database connection TDD suite
│   │   └── seeds/                # Mockup datasets & database seeder logic
│   │       ├── hospital.go       # Hospital mock data & transaction seeder
│   │       ├── hospital_test.go  # Seed data unit tests
│   │       └── seeder.go         # Seeder coordinator
│   ├── handlers/
│   │   ├── health_handler.go     # Health & probe HTTP handlers
│   │   ├── health_handler_test.go# Handler unit tests (mocked DB)
│   │   └── routes.go             # Route registration & grouping
│   ├── middleware/
│   │   ├── cors.go               # CORS middleware
│   │   └── cors_test.go          # Middleware unit tests
│   ├── models/                   # Domain entities and data structures
│   └── repository/               # Database access & query layer
├── migrations/                   # SQL migration scripts (.up.sql & .down.sql)
├── nginx/                        # Nginx reverse proxy configuration
│   ├── default.conf              # Reverse proxy, upstream, HTTP/HTTPS rules
│   └── ssl/                      # SSL certificate directory (Git-ignored)
│       ├── .gitkeep              # Tracks directory structure in Git
│       ├── generate-cert.sh      # Script to generate local SSL certificates
│       ├── server.crt            # Public SSL certificate (generated)
│       └── server.key            # Private SSL key (generated)
├── Dockerfile                    # Multi-stage Docker build
├── docker-compose.yml            # Docker Compose service definition
├── .dockerignore                 # Excluded build context files
├── .env.example                  # Template for environment variables
├── go.mod                        # Go module definitions
├── go.sum                        # Dependency checksums
└── README.md
```

---

## Prerequisites

- **Docker**: `20.10+` (or Docker Desktop)
- **Docker Compose**: `v2+`
- **Go**: `1.25.0` or higher *(only if developing/running outside Docker)*
- **OpenSSL**: *(used to generate local SSL certificates)*
- **golang-migrate**: *(optional, for running CLI migrations manually)*

---

## Quick Start for Developers

For a fresh checkout, run these steps to get the entire stack running:

```bash
# 1. Enter repository
cd hospital-middleware

# 2. Copy environment file
cp .env.example .env

# 3. Generate local SSL certificates (runs via host openssl or Docker automatically)
chmod +x ./nginx/ssl/generate-cert.sh
./nginx/ssl/generate-cert.sh

# 4. Start all services
docker compose up -d --build

# 5. Verify health
curl -k https://localhost/health
# Expected: {"database":"connected","status":"healthy"}
```

> **Note for Windows users:**
> - Run `./nginx/ssl/generate-cert.sh` inside **Git Bash** or **WSL**.


---

## Getting Started

### 1. Configure Environment

Copy the example environment file:
```bash
cp .env.example .env
```

Review `.env` settings if needed (default values work out of the box for local development).

### 2. Generate SSL Certificates

Because private keys (`*.key`) and certificates (`*.crt`) are excluded from Git for security, each developer must generate them once locally:

```bash
chmod +x ./nginx/ssl/generate-cert.sh
./nginx/ssl/generate-cert.sh
```

This generates `server.crt` and `server.key` inside `nginx/ssl/` with Subject Alternative Names (SAN) supporting `localhost` and `127.0.0.1`. *(If OpenSSL is not installed on the host, the script will automatically fallback to using Docker).*

### 3. Run with Docker Compose

#### Start services:
```bash
docker compose up -d --build
```

#### Check container health status:
```bash
docker compose ps
```
*(All services `postgres`, `hospital-middleware-api`, and `nginx` should show status **Up (healthy)**).*

#### View logs:
```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f hospital-middleware-api
docker compose logs -f nginx
docker compose logs -f postgres
```

#### Stop services:
```bash
docker compose down
```

---

### 4. Run Locally (Go Development)

If developing Go code locally without running the API container:

1. **Start only PostgreSQL**:
   ```bash
   docker compose up -d postgres
   ```

2. **Download dependencies**:
   ```bash
   go mod download
   ```

3. **Run the API server**:
   ```bash
   go run ./cmd/api
   ```

4. **Verify direct connection**:
   ```bash
   curl http://localhost:8080/health
   ```

---

## API Endpoints & Testing

### Endpoints

| Method | Endpoint | Description | Response Example |
| :--- | :--- | :--- | :--- |
| `GET` | `/health` | Readiness probe & PostgreSQL DB check | `{"database":"connected","status":"healthy"}` |
| `GET` | `/api/v1/health`| API v1 scoped database check | `{"database":"connected","status":"healthy"}` |

### Testing via curl

#### 1. Via HTTPS (Port 443 - Recommended)
> *Note: Use `-k` (or `--insecure`) because the SSL certificate is self-signed.*
```bash
# Health & Database connection test
curl -k https://localhost/health
```

#### 2. Via HTTP Auto-Redirect (Port 80 -> 443)
```bash
# Inspect 301 Redirect response
curl -i http://localhost/health

# Follow redirect automatically (-L)
curl -k -L http://localhost/health
```

#### 3. Direct to API Service (Port 8080 - Debugging)
```bash
curl http://localhost:8080/health
```

---

## Database Migrations

Database schema migrations are managed using [golang-migrate](https://github.com/golang-migrate/migrate) and stored in the [`migrations/`](migrations/) directory.

### 1. Automatic Migrations on Startup
By default, the application embeds all SQL migration files in the binary using Go's `embed.FS` and applies pending migrations automatically when the API starts (`DB_AUTO_MIGRATE=true` in `.env`).

To disable auto-migrations in production or CI/CD pipelines, set:
```env
DB_AUTO_MIGRATE=false
```

### 2. Manual CLI Migrations

Install the `migrate` CLI tool:
```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

#### Create a new migration:
```bash
migrate create -ext sql -dir migrations -seq <migration_name>
```
*Example:*
```bash
migrate create -ext sql -dir migrations -seq create_staff_table
```

#### Apply pending migrations (Up):
```bash
migrate -path migrations -database "postgres://admin:postgres@localhost:5432/hospital_middleware_db?sslmode=disable" up
```

#### Rollback last migration (Down):
```bash
migrate -path migrations -database "postgres://admin:postgres@localhost:5432/hospital_middleware_db?sslmode=disable" down 1
```

#### Check current version:
```bash
migrate -path migrations -database "postgres://admin:postgres@localhost:5432/hospital_middleware_db?sslmode=disable" version
```

---

## Database Seeder (Mock Data)

A dedicated Go Seeder CLI is provided under [`cmd/seed/`](cmd/seed/) to inject realistic mockup data into the database for local development and testing, keeping the `migrations/` directory clean and free of test-data pollution.

### Features
- **Production Safeguard:** Automatically prevents execution if `GIN_MODE=release` or `APP_ENV=production`, unless overridden with the `-force` flag.
- **Idempotency:** Utilizes `ON CONFLICT (hn) DO UPDATE` so seeders can be re-run safely without primary or unique key violations.
- **Single Transaction:** All inserts for each table run atomically inside a database transaction (`BeginTx`).

### Usage

Ensure the PostgreSQL database container or instance is running (`docker compose up -d postgres`), then run:

#### 1. Seed All Tables (Default):
```bash
go run ./cmd/seed
```

#### 2. Seed Specific Table:
```bash
# Seed only the hospital table
go run ./cmd/seed -table=hospital
```

#### 3. Seed in Production (Bypass Safeguard):
```bash
go run ./cmd/seed -table=hospital -force
```

#### 4. View Available CLI Flags:
```bash
go run ./cmd/seed -help
```

---

## Testing (TDD)

Run the full Go test suite (unit tests and database integration tests):

```bash
go test -v ./...
```

---

## Environment Variables

| Variable | Default | Description |
| :--- | :--- | :--- |
| `PORT` | `8080` | Port for the Go HTTP server |
| `GIN_MODE` | `debug` | Gin framework mode (`debug` or `release`) |
| `NGINX_PORT` | `80` | Host port for Nginx HTTP (redirects to HTTPS) |
| `NGINX_SSL_PORT` | `443` | Host port for Nginx HTTPS (SSL/TLS) |
| `DB_HOST` | `localhost` | PostgreSQL host (`postgres` when inside Docker) |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `admin` | PostgreSQL username |
| `DB_PASSWORD` | `postgres` | PostgreSQL password |
| `DB_NAME` | `hospital_middleware_db` | PostgreSQL database name |
| `DB_SSLMODE` | `disable` | PostgreSQL SSL mode (`disable`, `require`, etc.) |
| `DB_AUTO_MIGRATE` | `true` | Automatically run migrations on startup (`true` / `false`) |

---

## Troubleshooting

### 1. Nginx fails to start: missing `server.crt` or `server.key`
SSL certificate files are git-ignored for security. Generate them by running:
```bash
./nginx/ssl/generate-cert.sh
docker compose up -d --force-recreate nginx
```

### 2. SSL certificate warning in browser or curl (`certificate verify failed`)
Because the certificate is self-signed for local development:
- **curl**: Pass the `-k` (or `--insecure`) flag.
- **Chrome / Firefox**: Click "Advanced" -> "Proceed to localhost (unsafe)".

### 3. Dependency version conflicts (`requires go >= 1.26`)
Avoid running `go get -u` across all packages, as it upgrades transitive dependencies to unreleased Go versions. Instead, install specific packages:
```bash
go get <package-name>
go mod tidy
```
