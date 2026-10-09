# Hospital Middleware

A Go-based API middleware service built with [Gin Web Framework](https://github.com/gin-gonic/gin), featuring PostgreSQL database integration and an Nginx reverse proxy with SSL/TLS (HTTPS) support.

---

## Table of Contents

- [Features](#features)
- [Architecture & Services](#architecture--services)
- [Project Structure](#project-structure)
- [Prerequisites](#prerequisites)
- [Quick Start with Docker Compose (3 Minutes)](#quick-start-with-docker-compose-3-minutes)
- [Docker Compose Guide](#docker-compose-guide)
  - [1. Configure Environment](#1-configure-environment)
  - [2. Generate SSL/TLS Certificates](#2-generate-ssltls-certificates)
  - [3. Start Services & Verify Health](#3-start-services--verify-health)
  - [4. Seed Database Mockup Data](#4-seed-database-mockup-data)
  - [5. Docker Compose Daily Cheatsheet](#5-docker-compose-daily-cheatsheet)
  - [6. Connecting to PostgreSQL (GUI & CLI)](#6-connecting-to-postgresql-gui--cli)
  - [7. Customizing Exposed Ports](#7-customizing-exposed-ports)
- [Local Go Development (Without Docker API)](#local-go-development-without-docker-api)
- [API Endpoints & Testing](#api-endpoints--testing)
  - [Endpoints](#endpoints)
  - [Interactive API Documentation (Swagger UI)](#interactive-api-documentation-swagger-ui)
  - [Request Validation Rules](#request-validation-rules)
  - [Testing via curl (End-to-End Walkthrough)](#testing-via-curl-end-to-end-walkthrough)
  - [Error Handling (RFC 7807 Problem Details)](#error-handling-rfc-7807-problem-details)
- [Database Migrations](#database-migrations)
  - [1. Automatic Migrations on Startup](#1-automatic-migrations-on-startup)
  - [2. Migration Files Overview](#2-migration-files-overview)
  - [3. Manual CLI Migrations](#3-manual-cli-migrations)
- [Database Seeder Deep Dive](#database-seeder-deep-dive)
- [Testing (TDD Suite)](#testing-tdd-suite)
- [Environment Variables](#environment-variables)
- [Troubleshooting & FAQ](#troubleshooting--faq)

---

## Features

- **Gin HTTP Server**: Fast and lightweight REST API framework with graceful shutdown (SIGINT/SIGTERM trapping) and HTTP server timeout protection (`ReadTimeout`, `WriteTimeout`, `IdleTimeout`).
- **Nginx Reverse Proxy & HTTP/2**: Production-ready reverse proxy with HTTP-to-HTTPS redirection via `308 Permanent Redirect` (preserving HTTP request method and payload), upstream keepalive connection pooling, and HTTP/2 protocol support.
- **SSL/TLS (HTTPS)**: Preconfigured Nginx reverse proxy with SAN-enabled SSL certificates supporting `localhost`, IPv4 `127.0.0.1`, and IPv6 `::1`.
- **Security Hardening**: Nginx version banner hidden (`server_tokens off`), strict HTTP security headers (`X-Content-Type-Options`, `X-Frame-Options`, `X-XSS-Protection`, `Referrer-Policy`), and strict SSL file permissions (`600` for private key, `644` for certificate).
- **PostgreSQL 18 Integration**: Database layer with connection pooling (`DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, lifetime & idle time tuning), health check probes, and automated schema migrations.
- **JWT Authentication & Session Persistence**: Secure session management using HMAC-SHA256 (`HS256`) JWT tokens stored in the `staff_session` database table and dispatched to clients via `HttpOnly`, `SameSite=Lax`, and `Secure` cookies.
- **Strict Password & Identity Policy**: Alphanumeric username validation, comprehensive password policy (8–32 chars, ASCII English only, lowercase, uppercase, digit, and special symbol), and password hashing via `bcrypt`.
- **RFC 7807 Problem Details**: Centralized error handling returning standardized `application/problem+json` error responses with distinct error URIs and descriptive titles.
- **Dynamic CORS Middleware**: Origin whitelist filtering (`CORS_ALLOWED_ORIGINS`), credential forwarding (`Access-Control-Allow-Credentials: true`), and preflight caching (`CORS_MAX_AGE_SECONDS`).
- **Dockerized**: Multi-stage minimal non-root Alpine container with automated healthcheck orchestration on `bridge-network`.
- **Database Migrations**: Embedded migrations (000001 through 000004) using `golang-migrate` and Go `embed.FS`.
- **Database Seeder**: Standalone CLI tool (`cmd/seed`) with environment safety guards and idempotent single-transaction seeding.
- **Interactive Swagger / OpenAPI 2.0**: Embedded Swagger UI served at `/swagger/*any` using `swaggo/gin-swagger`, providing interactive endpoint documentation, schemas, and live testing in both HTTP and HTTPS modes.

---

## Architecture & Services

When running with Docker Compose, three services are orchestrated in the `bridge-network` bridge network:

```
[Client / Browser / curl]
         │
         ├─────────────────────────────────────────┐
         ▼ (Port 80 / 443)                         ▼ (Port 8080 - Optional Debug)
┌─────────────────────────────────┐       ┌─────────────────────────────────┐
│              Nginx              │       │          Go API Service         │
│              (nginx)            │──────>│     (hospital-middleware-api)   │
│  - Port 80 (HTTP -> 308 HTTPS)  │       │  - Port 8080                    │
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
| **`nginx`** | `nginx` | `80`, `443` | `80`, `443` | Reverse proxy, SSL termination, and HTTP 308 redirect |
| **`hospital-middleware-api`** | `hospital-middleware-api` | `8080` | `8080` | Go Gin REST API application |
| **`postgres`** | `postgres` | `5432` | `5432` | PostgreSQL 18.4 database |

---

## Project Structure

```text
hospital-middleware/
├── cmd/
│   ├── api/
│   │   ├── main.go                       # Application entrypoint, DI, server timeouts & graceful shutdown
│   │   └── main_test.go                  # API route integration tests (mocked & live DB)
│   └── seed/
│       └── main.go                       # Standalone database seeder CLI
├── docs/                                 # Auto-generated Swagger/OpenAPI documentation
│   ├── docs.go                           # Go package registering Swagger specification
│   ├── swagger.json                      # OpenAPI specification in JSON
│   └── swagger.yaml                      # OpenAPI specification in YAML
├── internal/
│   ├── auth/                             # Authentication & JWT token service (HS256)
│   │   ├── jwt.go                        # Token generator & claims validator
│   │   └── jwt_test.go                   # JWT unit tests
│   ├── config/
│   │   ├── config.go                     # Configuration loader, environment validator, and DSN builder
│   │   └── config_test.go                # Config unit tests
│   ├── database/
│   │   ├── migrate.go                    # Embedded migration executor (golang-migrate)
│   │   ├── migrate_test.go               # Migration runner unit tests
│   │   ├── postgres.go                   # PostgreSQL connection pool & management
│   │   ├── postgres_test.go              # Database connection TDD suite
│   │   └── seeds/                        # Mockup datasets & database seeder logic
│   │       ├── hospital.go               # Hospital mock data & transaction seeder
│   │       ├── hospital_test.go          # Seed data unit tests
│   │       └── seeder.go                 # Seeder coordinator
│   ├── handlers/
│   │   ├── health_handler.go             # Health & readiness probe HTTP handlers
│   │   ├── health_handler_test.go        # Health handler unit tests (mocked DB)
│   │   ├── staff_handler.go              # Staff HTTP handler (/staff/create, /staff/login) & password validator
│   │   ├── staff_handler_test.go         # Staff handler unit tests
│   │   └── routes.go                     # Route registration & global middleware attachment
│   ├── middleware/
│   │   ├── cors.go                       # Dynamic CORS middleware (origin whitelist, credentials, max-age)
│   │   ├── cors_test.go                  # CORS unit tests
│   │   ├── error_handler.go              # Centralized RFC 7807 Problem Details error handler
│   │   └── error_handler_test.go         # Error handler unit tests
│   ├── models/                           # Domain entities and data structures
│   │   ├── hospital.go                   # Hospital domain entity
│   │   ├── staff.go                      # Staff domain entity
│   │   └── staff_session.go              # Staff session domain entity
│   ├── repository/                       # Database access & query layer (Data persistence)
│   │   ├── hospital_repository.go        # Hospital data access implementation
│   │   ├── hospital_repository_test.go   # Hospital repository unit & integration tests
│   │   ├── staff_repository.go           # Staff data access implementation
│   │   ├── staff_repository_test.go      # Staff repository unit & integration tests
│   │   ├── staff_session_repository.go   # Staff session data access implementation
│   │   └── staff_session_repository_test.go # Staff session repository unit & integration tests
│   ├── response/                         # Reusable JSend / Custom Envelope response package
│   │   ├── response.go                   # Generic response envelope & helpers
│   │   └── response_test.go              # Response envelope unit tests
│   └── service/                          # Business logic layer (Domain rules)
│       ├── staff_service.go              # Staff business logic (bcrypt, JWT issuance, session persistence)
│       └── staff_service_test.go         # Staff service unit tests
├── migrations/                           # SQL migration scripts (.up.sql & .down.sql)
│   ├── 000001_create_hospital_table.up.sql
│   ├── 000001_create_hospital_table.down.sql
│   ├── 000002_create_staff_table.up.sql
│   ├── 000002_create_staff_table.down.sql
│   ├── 000003_create_patient_table.up.sql
│   ├── 000003_create_patient_table.down.sql
│   ├── 000004_create_staff_session_table.up.sql
│   ├── 000004_create_staff_session_table.down.sql
│   └── migrations.go                     # Go embed.FS binding for migrations
├── nginx/                                # Nginx reverse proxy configuration
│   ├── default.conf                      # Reverse proxy, upstream keepalive, HTTP 308, security headers
│   └── ssl/                              # SSL certificate directory (Git-ignored)
│       ├── .gitkeep                      # Tracks directory structure in Git
│       ├── generate-cert.sh              # Script to generate local SSL certificates (SAN: IPv4 & IPv6)
│       ├── server.crt                    # Public SSL certificate (generated)
│       └── server.key                    # Private SSL key (generated)
├── Dockerfile                            # Multi-stage Docker build
├── docker-compose.yml                    # Docker Compose service definition
├── .dockerignore                         # Excluded build context files
├── .env.example                          # Template for environment variables
├── go.mod                                # Go module definitions
├── go.sum                                # Dependency checksums
└── README.md
```

---

## Prerequisites

- **For Docker Compose (Recommended)**:
  - **Docker**: `20.10+` (or Docker Desktop)
  - **Docker Compose**: `v2+`
  - *(That's all! You do **not** need Go, PostgreSQL, or OpenSSL installed locally on your host machine to run, seed, or test the stack).*
- **For Host Development (Go native outside Docker)**:
  - **Go**: `1.25.0` or higher
  - **OpenSSL**: *(used by `generate-cert.sh` on host; Docker fallback is automatic)*
  - **swag CLI**: *(optional, for regenerating Swagger docs)*: `go install github.com/swaggo/swag/cmd/swag@latest`
  - **golang-migrate**: *(optional, for running manual migrations via CLI)*

---

## Quick Start with Docker Compose (3 Minutes)

Get the entire stack (PostgreSQL + API + Nginx SSL + Seeded Mock Data) up and running in **one copy-paste command**:

```bash
# One-liner quick start:
cp .env.example .env && \
chmod +x ./nginx/ssl/generate-cert.sh && ./nginx/ssl/generate-cert.sh && \
docker compose up -d --build && \
docker run --rm --network bridge-network -v "$PWD":/app -w /app --env-file .env -e DB_HOST=postgres golang:1.25-alpine go run ./cmd/seed
```

Or follow the step-by-step instructions below:

### Step 1: Clone Repository & Configure Environment
```bash
git clone https://github.com/scale10n/hospital-middleware.git
cd hospital-middleware
cp .env.example .env
```
> The default variables in `.env.example` work out of the box for local Docker Compose development.

### Step 2: Generate Local SSL/TLS Certificates
Nginx mounts `./nginx/ssl` to terminate HTTPS. Because private keys and certificates are excluded from Git (`.gitignore`), generate them once:
```bash
chmod +x ./nginx/ssl/generate-cert.sh
./nginx/ssl/generate-cert.sh
```
> **Tip:** If `openssl` is not installed on your host system, the script automatically uses Docker (`alpine`) to generate SAN-enabled certificates. Windows users can run this in **Git Bash** or **WSL**.

### Step 3: Start Services with Docker Compose
```bash
docker compose up -d --build
```
This builds and launches the containers in sequence with automated health checks:
1. `postgres` (PostgreSQL 18.4) starts on port `5432`.
2. `hospital-middleware-api` waits for PostgreSQL, auto-applies database migrations (`DB_AUTO_MIGRATE=true`), and starts on port `8080`.
3. `nginx` waits for the API health probe, binds ports `80` and `443`, and terminates SSL.

Check container status:
```bash
docker compose ps
```
*(All 3 services will show status `Up (healthy)` after ~10-15 seconds).*

### Step 4: Seed Database with Mockup Data (Crucial Step!)
When started for the first time, the database tables are migrated but empty. Running the seeder inserts **10 hospitals** (`HOSP001` - `HOSP010`) and **45 patients** required for staff authentication and patient search:

```bash
# Run seeder via Docker (No local Go installation required!)
docker run --rm --network bridge-network -v "$PWD":/app -w /app \
  --env-file .env -e DB_HOST=postgres \
  golang:1.25-alpine go run ./cmd/seed
```
*(Or if you have Go installed on your host machine: `go run ./cmd/seed`)*

### Step 5: Verify Health & Explore Swagger UI
- **Health Check**:
  ```bash
  curl -k https://localhost/health
  # Expected: {"database":"connected","status":"healthy"}
  ```
- **Interactive Swagger UI**: Open in your browser:
  - **Via HTTPS (Nginx Gateway)**: [https://localhost/swagger/index.html](https://localhost/swagger/index.html)
  - **Via HTTP (Direct API Debug)**: [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)

---

## Docker Compose Guide

### 1. Configure Environment

The project uses `.env` for container orchestration. Key settings:

| Variable | Default | Purpose in Docker Compose |
| :--- | :--- | :--- |
| `DB_USER` | `admin` | PostgreSQL superuser username |
| `DB_PASSWORD` | `your_secure_password` | PostgreSQL superuser password (matches in db & api) |
| `DB_NAME` | `hospital_middleware_db` | Application database name |
| `PORT` | `8080` | Internal API port |
| `NGINX_PORT` | `80` | Host port for HTTP (redirects 308 to HTTPS) |
| `NGINX_SSL_PORT` | `443` | Host port for HTTPS gateway |
| `JWT_SECRET` | *(32+ chars)* | HMAC-SHA256 secret for staff session tokens |
| `DB_AUTO_MIGRATE`| `true` | Automatically runs embedded migrations on API startup |

### 2. Generate SSL/TLS Certificates

The script `./nginx/ssl/generate-cert.sh` creates a 2048-bit RSA certificate valid for 365 days with Subject Alternative Names (SAN) supporting:
- `localhost`
- IPv4 loopback: `127.0.0.1`
- IPv6 loopback: `::1`

```bash
# Default (localhost, 365 days):
./nginx/ssl/generate-cert.sh

# Custom domain and duration:
./nginx/ssl/generate-cert.sh api.local 730
```

### 3. Start Services & Verify Health

```bash
# Start all containers in detached mode:
docker compose up -d --build

# Verify container health status:
docker compose ps
```

Example healthy output:
```text
NAME                      IMAGE                                         COMMAND                  SERVICE                   CREATED          STATUS                    PORTS
hospital-middleware-api   hospital-middleware-hospital-middleware-api   "/app/api"               hospital-middleware-api   10 seconds ago   Up 9 seconds (healthy)    0.0.0.0:8080->8080/tcp
nginx                     nginx:alpine                                  "/docker-entrypoint.…"   nginx                     10 seconds ago   Up 8 seconds (healthy)    0.0.0.0:80->80/tcp, 0.0.0.0:443->443/tcp
postgres                  postgres:18.4                                 "docker-entrypoint.s…"   postgres                  10 seconds ago   Up 10 seconds (healthy)   0.0.0.0:5432->5432/tcp
```

### 4. Seed Database Mockup Data

Without seed data, calling `POST /staff/create` with `hospital: "HOSP001"` will fail with `400 Bad Request` (`hospital not found`).

Run the seeder with one of two options:

- **Option A (Via Docker - No Go needed on host)**:
  ```bash
  docker run --rm --network bridge-network -v "$PWD":/app -w /app \
    --env-file .env -e DB_HOST=postgres \
    golang:1.25-alpine go run ./cmd/seed
  ```
- **Option B (Via Host Go)**:
  ```bash
  go run ./cmd/seed
  ```

Seeder features:
- **Idempotent**: Can be re-run safely at any time (`ON CONFLICT (hn) DO UPDATE`).
- **Atomic**: Runs inside a single database transaction.
- **Selective Seeding**: Pass `-table=hospital` or `-table=patient` to seed specific datasets.

### 5. Docker Compose Daily Cheatsheet

| Action | Command |
| :--- | :--- |
| **Start / Build stack** | `docker compose up -d --build` |
| **Stop stack (preserve data)** | `docker compose down` |
| **Stop & Delete database volume** | `docker compose down -v` |
| **View logs (all services)** | `docker compose logs -f` |
| **View API logs only** | `docker compose logs -f hospital-middleware-api` |
| **View Nginx logs only** | `docker compose logs -f nginx` |
| **View Postgres logs only** | `docker compose logs -f postgres` |
| **Rebuild API after code change** | `docker compose up -d --build hospital-middleware-api` |
| **Restart single service** | `docker compose restart <service-name>` |
| **Enter Postgres psql CLI** | `docker compose exec -it postgres psql -U admin -d hospital_middleware_db` |
| **Enter API container shell** | `docker compose exec -it hospital-middleware-api /bin/sh` |
| **Reload Nginx configuration** | `docker compose exec nginx nginx -s reload` |

### 6. Connecting to PostgreSQL (GUI & CLI)

The PostgreSQL container exposes port `5432` on `localhost`. You can connect using your favorite database GUI client (DBeaver, TablePlus, DataGrip, pgAdmin) or CLI:

- **Host**: `localhost` (or `127.0.0.1`)
- **Port**: `5432` *(or the value of `DB_PORT` in `.env`)*
- **Database**: `hospital_middleware_db`
- **Username**: `admin` *(or value of `DB_USER` in `.env`)*
- **Password**: `your_secure_password` *(matches `DB_PASSWORD` in `.env`)*
- **SSL Mode**: `disable`

Or open an interactive SQL prompt directly:
```bash
docker compose exec -it postgres psql -U admin -d hospital_middleware_db
```

### 7. Customizing Exposed Ports

If ports `80`, `443`, `8080`, or `5432` are already used by other services on your machine, customize them in `.env`:

```env
# Avoid port 80/443 conflict on host:
NGINX_PORT=8088
NGINX_SSL_PORT=8443

# Avoid port 8080/5432 conflict:
PORT=8888
DB_PORT=5433
```
Then restart with `docker compose up -d`. You can then access HTTPS via `https://localhost:8443`.

---

## Local Go Development (Without Docker API)

If you prefer to edit and run the Go code natively with live reload on your host machine while keeping PostgreSQL inside Docker:

1. **Start only PostgreSQL**:
   ```bash
   docker compose up -d postgres
   ```

2. **Ensure Go is available in your PATH**:
   ```bash
   export PATH=$PATH:/usr/local/go/bin:~/go/bin
   go version
   ```

3. **Install dependencies**:
   ```bash
   go mod download
   ```

4. **Seed database mock data**:
   ```bash
   go run ./cmd/seed
   ```

5. **Run the API server**:
   ```bash
   go run ./cmd/api
   ```

6. **Verify direct connection**:
   ```bash
   curl http://localhost:8080/health
   ```

---

## API Endpoints & Testing

### Endpoints

| Method | Endpoint | Description | Status Code | Content-Type |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/swagger/*any` | Interactive Swagger UI API documentation | `200 OK` | `text/html` |
| `GET` | `/health` | Readiness probe & PostgreSQL DB check | `200 OK` | `application/json` |
| `POST` | `/staff/create` | Validate hospital HN, create staff member, hash password, persist session & issue JWT | `201 Created` | `application/json` |
| `POST` | `/staff/login` | Authenticate staff credentials, persist session & issue JWT | `200 OK` | `application/json` |
| `POST` | `/patient/search` | Search patients within staff's hospital matching all provided criteria (CookieAuth) | `200 OK` | `application/json` |
| `GET` | `/patient/search/{id}` | Retrieve specific patient details by Thai National ID (13 digits) or Passport ID within staff's hospital (CookieAuth) | `200 OK` | `application/json` |

---

### Interactive API Documentation (Swagger UI)

The API includes embedded, self-contained interactive documentation powered by Swagger UI.

#### Access Swagger UI:
- **Direct (Local Dev)**: [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)
- **Via Nginx (HTTPS)**: [https://localhost/swagger/index.html](https://localhost/swagger/index.html)
- **Raw JSON Spec**: `http://localhost:8080/swagger/doc.json`

#### CLI Tool Installation (`swag`):
To generate or update Swagger documentation from code annotations, install the `swag` CLI tool:
```bash
# Ensure Go bin directory is in PATH
export PATH=$PATH:$HOME/go/bin

# Install swag CLI
go install github.com/swaggo/swag/cmd/swag@latest

# Verify installation
swag --version
```

#### Regenerating Swagger Docs:
Whenever you add or update API annotations in `cmd/api/main.go` or `internal/handlers/*.go`, run:
```bash
swag init -g cmd/api/main.go -o ./docs --parseDependency --parseInternal
```

The generated files are saved into the [`docs/`](docs/) directory:
- `docs/docs.go`: Go source code registering the specification with Swagger at runtime.
- `docs/swagger.json`: OpenAPI 2.0 JSON specification.
- `docs/swagger.yaml`: OpenAPI 2.0 YAML specification.

---

### Request Validation Rules

#### `POST /staff/create` Request Payload
```json
{
  "username": "somchai",
  "password": "Password123!",
  "hospital": "HOSP001"
}
```

- **`username`** *(required, string)*:
  - Must be strictly alphanumeric (`[a-zA-Z0-9]`).
  - Thai characters, spaces, and punctuation/special characters are rejected.
- **`password`** *(required, string)*:
  - Length: between 8 and 32 characters.
  - Allowed characters: English letters, numbers, and allowed ASCII symbols (`!@#$%^&*()-_=+[]{}|;:'",.<>/?`~\\`).
  - Character requirements:
    - At least 1 lowercase letter (`a-z`)
    - At least 1 uppercase letter (`A-Z`)
    - At least 1 digit (`0-9`)
    - At least 1 special character
  - Rejects Thai characters, emojis, and non-ASCII characters.
- **`hospital`** *(required, string)*:
  - Must match an existing hospital HN in the database (e.g. `HOSP001`).

#### `POST /staff/login` Request Payload
```json
{
  "username": "somchai",
  "password": "Password123!",
  "hospital": "HOSP001"
}
```

The validation rules for `username`, `password`, and `hospital` are identical to `POST /staff/create`:
- **`username`** *(required, string)*:
  - Must be strictly alphanumeric (`[a-zA-Z0-9]`).
  - Thai characters, spaces, and punctuation/special characters are rejected (`400 Bad Request`).
- **`password`** *(required, string)*:
  - Length: between 8 and 32 characters.
  - Allowed characters: English letters, numbers, and allowed ASCII symbols (`!@#$%^&*()-_=+[]{}|;:'",.<>/?`~\\`).
  - Character requirements:
    - At least 1 lowercase letter (`a-z`)
    - At least 1 uppercase letter (`A-Z`)
    - At least 1 digit (`0-9`)
    - At least 1 special character
  - Rejects Thai characters, emojis, spaces, and non-ASCII characters (`400 Bad Request`).
- **`hospital`** *(required, string)*:
  - Must match an existing hospital HN in the database (e.g. `HOSP001`). If not found, returns `400 Bad Request` (`hospital not found`).

#### `POST /patient/search` Query Parameters & Payload
Accepts optional query parameters and/or JSON body:
- **`national_id`** *(optional, string)*: Thai Citizen National ID.
- **`passport_id`** *(optional, string)*: Passport number.
- **`first_name`** *(optional, string)*: Patient first name in Thai or English (supports case-insensitive partial match via PostgreSQL `ILIKE`).
- **`middle_name`** *(optional, string)*: Patient middle name in Thai or English (supports case-insensitive partial match via PostgreSQL `ILIKE`).
- **`last_name`** *(optional, string)*: Patient last name in Thai or English (supports case-insensitive partial match via PostgreSQL `ILIKE`).
- **`date_of_birth`** *(optional, string)*: Format `YYYY-MM-DD` (e.g. `1990-05-15`).
- **`phone_number`** *(optional, string)*: Patient contact number.
- **`email`** *(optional, string)*: Patient email address.

**Rules & Constraints:**
- **Authentication**: Requires valid JWT session cookie `session_token` issued at staff login (`CookieAuth`). Rejects unauthorized requests with `401 Unauthorized`.
- **Hospital Isolation**: Results are strictly filtered to the authenticated staff member's hospital (`WHERE patient.hospital_id = staff.hospital_id`).
- **Search Criteria**: At least one of the 8 search criteria above must be non-empty (rejects empty search queries with `400 Bad Request`).
- **AND Logic**: Returns patients matching ALL provided filter criteria.
- **Response Format**: JSend success envelope with patient details and total count (`total`).

#### `GET /patient/search/{id}` Path Parameters
Retrieves a single patient record by identity number within the authenticated staff's hospital:
- **`id`** *(required, path parameter)*: Thai National ID (strictly 13 digits, e.g. `1100501234567`) OR Foreign Passport ID (6 to 20 alphanumeric characters, case-insensitive, e.g. `US556677889`). Patient UUIDs are not accepted.

**Rules & Constraints:**
- **Authentication**: Requires valid JWT session cookie `session_token` (`CookieAuth`). Rejects unauthorized requests with `401 Unauthorized`.
- **Hospital Isolation**: Results are strictly filtered to the authenticated staff member's hospital (`WHERE p.hospital_id = staff.hospital_id AND (p.national_id = $2 OR p.passport_id ILIKE $2)`). Patients from other hospitals return `404 Not Found`.
- **Validation**: Rejects invalid identifier formats (must be 13 numeric digits or 6-20 alphanumeric characters) with `400 Bad Request`.
- **Response Format**: JSend success envelope containing all 14 patient fields (`id`, `patient_hn`, `first_name_th`, `middle_name_th`, `last_name_th`, `first_name_en`, `middle_name_en`, `last_name_en`, `date_of_birth`, `national_id`, `passport_id`, `phone_number`, `email`, `gender`).

---

### Testing via curl (End-to-End Walkthrough)

#### 1. Via HTTPS (Port 443 - Recommended)
> *Note: Use `-k` (or `--insecure`) because the SSL certificate is self-signed.*

```bash
# Health & Database connection test
curl -k https://localhost/health

# Create Staff Member (JSend / Custom Envelope response & HttpOnly cookie)
curl -k -i -X POST https://localhost/staff/create \
  -H "Content-Type: application/json" \
  -d '{
    "username": "somchai",
    "password": "Password123!",
    "hospital": "HOSP001"
  }'
```

**HTTP 201 Created Response:**
- Header: `Set-Cookie: session_token=<jwt_token>; Path=/; Max-Age=86400; HttpOnly; SameSite=Lax; [Secure]`
- Response Body:
```json
{
  "status": "success",
  "message": "staff created successfully",
  "data": {
    "staff": {
      "id": "76ec965b-bf50-48b4-82a4-7935f8c6ebf7",
      "username": "somchai",
      "hospital_hn": "HOSP001",
      "hospital_name": "โรงพยาบาลศิริราช (Siriraj Hospital)",
      "created_at": "2026-10-08T17:34:38.123456Z"
    },
    "session": {
      "expires_at": "2026-10-09T17:34:38.123456Z"
    }
  }
}
```

```bash
# Staff Login (JSend / Custom Envelope response & HttpOnly cookie)
curl -k -i -X POST https://localhost/staff/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "somchai",
    "password": "Password123!",
    "hospital": "HOSP001"
  }'
```

**HTTP 200 OK Response:**
- Header: `Set-Cookie: session_token=<jwt_token>; Path=/; Max-Age=86400; HttpOnly; SameSite=Lax; [Secure]`
- Response Body:
```json
{
  "status": "success",
  "message": "login successful",
  "data": {
    "staff": {
      "id": "76ec965b-bf50-48b4-82a4-7935f8c6ebf7",
      "username": "somchai",
      "hospital_hn": "HOSP001",
      "hospital_name": "โรงพยาบาลศิริราช (Siriraj Hospital)",
      "created_at": "2026-10-08T17:34:38.123456Z"
    },
    "session": {
      "expires_at": "2026-10-09T17:34:38.123456Z"
    }
  }
}
```

```bash
# Patient Search (requires session_token cookie from login)
curl -k -i -X POST "https://localhost/patient/search?first_name=Somchai" \
  --cookie "session_token=<jwt_token>"
```

**HTTP 200 OK Response:**
```json
{
  "status": "success",
  "message": "patients retrieved successfully",
  "data": {
    "patients": [
      {
        "id": "c1f728ea-a312-4f35-905e-8b1d9bf5b012",
        "patient_hn": "P00001",
        "first_name_th": "สมชาย",
        "middle_name_th": "วิชัย",
        "last_name_th": "ใจดี",
        "first_name_en": "Somchai",
        "middle_name_en": "Wichai",
        "last_name_en": "Jaidee",
        "date_of_birth": "1985-04-12",
        "national_id": "1100501234567",
        "passport_id": "",
        "phone_number": "0812345678",
        "email": "somchai.jaidee@example.com",
        "gender": "M"
      }
    ],
    "total": 1
  }
}
```

```bash
# Patient Search by Identity (Thai National ID or Passport ID, requires session_token cookie from login)
# Example 1: Search by 13-digit Thai National ID
curl -k -i -X GET "https://localhost/patient/search/1100501234567" \
  --cookie "session_token=<jwt_token>"

# Example 2: Search by Foreign Passport ID
curl -k -i -X GET "https://localhost/patient/search/US556677889" \
  --cookie "session_token=<jwt_token>"
```

**HTTP 200 OK Response:**
```json
{
  "status": "success",
  "message": "patient retrieved successfully",
  "data": {
    "id": "c1f728ea-a312-4f35-905e-8b1d9bf5b012",
    "patient_hn": "P00001",
    "first_name_th": "สมชาย",
    "middle_name_th": "วิชัย",
    "last_name_th": "ใจดี",
    "first_name_en": "Somchai",
    "middle_name_en": "Wichai",
    "last_name_en": "Jaidee",
    "date_of_birth": "1985-04-12",
    "national_id": "1100501234567",
    "passport_id": "",
    "phone_number": "0812345678",
    "email": "somchai.jaidee@example.com",
    "gender": "M"
  }
}
```

#### 2. Via HTTP Auto-Redirect (Port 80 -> 443 via 308)
Nginx issues an **HTTP 308 Permanent Redirect**, which guarantees that clients preserve the HTTP method (`POST`) and request body when redirecting to HTTPS:
```bash
# Inspect 308 Permanent Redirect response
curl -i http://localhost/health

# Automatically follow redirect (-L) for POST request
curl -k -L -i -X POST http://localhost/staff/create \
  -H "Content-Type: application/json" \
  -d '{
    "username": "nurseann",
    "password": "Password123!",
    "hospital": "HOSP001"
  }'
```

#### 3. Direct to API Service (Port 8080 - Debugging)
```bash
# Health probe
curl http://localhost:8080/health

# Create Staff Member
curl -i -X POST http://localhost:8080/staff/create \
  -H "Content-Type: application/json" \
  -d '{
    "username": "somchai",
    "password": "Password123!",
    "hospital": "HOSP001"
  }'
```

---

### Error Handling (RFC 7807 Problem Details)

All API errors conform to the RFC 7807 specification with Content-Type `application/problem+json`:

#### 1. Hospital Not Found (`400 Bad Request`)
Returned when the requested hospital HN does not exist in the database:
```json
{
  "type": "/errors/hospital-not-found",
  "title": "Bad Request",
  "status": 400,
  "detail": "hospital not found",
  "instance": "/staff/create",
  "error": "hospital not found"
}
```

#### 2. Staff Already Exists (`409 Conflict`)
Returned when the staff username is already registered for this hospital:
```json
{
  "type": "/errors/staff-already-exists",
  "title": "Conflict",
  "status": 409,
  "detail": "staff already exists",
  "instance": "/staff/create",
  "error": "staff already exists"
}
```

#### 3. Request Validation Failure (`400 Bad Request`)
Returned when username format or password security policy rules fail:
```json
{
  "type": "/errors/validation-error",
  "title": "Bad Request",
  "status": 400,
  "detail": "Key: 'CreateStaffRequest.Password' Error:Field validation for 'Password' failed on the 'password' tag",
  "instance": "/staff/create",
  "error": "Key: 'CreateStaffRequest.Password' Error:Field validation for 'Password' failed on the 'password' tag"
}
```

#### 4. Invalid Credentials (`401 Unauthorized`)
Returned on `POST /staff/login` when the username is not found or password does not match:
```json
{
  "type": "/errors/unauthorized",
  "title": "Unauthorized",
  "status": 401,
  "detail": "invalid username or password",
  "instance": "/staff/login",
  "error": "invalid username or password"
}
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

### 2. Migration Files Overview

| Version | Migration Name | Description |
| :--- | :--- | :--- |
| `000001` | `create_hospital_table` | Hospital entity table (`id`, `hn`, `name`, timestamps) |
| `000002` | `create_staff_table` | Staff entity table with hospital foreign key & unique `(hospital_id, username)` |
| `000003` | `create_patient_table` | Patient entity table |
| `000004` | `create_staff_session_table` | Staff authentication session table (`staff_id`, `token`, `expires_at`) with foreign key cascade & indices |

### 3. Manual CLI Migrations

Install the `migrate` CLI tool:
```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

#### Create a new migration:
```bash
migrate create -ext sql -dir migrations -seq <migration_name>
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

## Database Seeder Deep Dive

A dedicated Go Seeder CLI is provided under [`cmd/seed/`](cmd/seed/) to inject realistic mockup data into PostgreSQL for local development and testing, keeping the `migrations/` directory clean and free of test-data pollution.

### Features
- **Production Safeguard:** Automatically prevents execution if `GIN_MODE=release` or `APP_ENV=production`, unless overridden with the `-force` flag.
- **Idempotency:** Utilizes `ON CONFLICT (hn) DO UPDATE` so seeders can be re-run safely without primary or unique key violations.
- **Single Transaction:** All inserts for each table run atomically inside a database transaction (`BeginTx`).

### Pre-Seeded Datasets for Testing

#### 1. Hospitals (10 Master Records)
| HN Code | Hospital Name (Thai & English) |
| :--- | :--- |
| `HOSP001` | โรงพยาบาลศิริราช (Siriraj Hospital) |
| `HOSP002` | โรงพยาบาลจุฬาลงกรณ์ สภากาชาดไทย (King Chulalongkorn Memorial Hospital) |
| `HOSP003` | โรงพยาบาลรามาธิบดี (Ramathibodi Hospital) |
| `HOSP004` | โรงพยาบาลกรุงเทพ (Bangkok Hospital) |
| `HOSP005` | โรงพยาบาลบำรุงราษฎร์ (Bumrungrad International Hospital) |
| `HOSP006` - `HOSP010` | Samitivej, Phramongkutklao, Rajavithi, Thammasat, Maharaj Nakorn Chiang Mai |

#### 2. Patients (45 Mock Records)
- **Thai Citizen Example**: National ID `1100501234567` (Somchai Jaidee, Hospital `HOSP001`)
- **Foreign Passport Example**: Passport ID `US556677889` (John Doe, Hospital `HOSP001`)

### Execution Commands

#### Option A: Running via Docker (No Go installed on host)
```bash
# Seed all tables (Hospitals + Patients):
docker run --rm --network bridge-network -v "$PWD":/app -w /app \
  --env-file .env -e DB_HOST=postgres \
  golang:1.25-alpine go run ./cmd/seed

# Seed only hospitals:
docker run --rm --network bridge-network -v "$PWD":/app -w /app \
  --env-file .env -e DB_HOST=postgres \
  golang:1.25-alpine go run ./cmd/seed -table=hospital

# Seed only patients:
docker run --rm --network bridge-network -v "$PWD":/app -w /app \
  --env-file .env -e DB_HOST=postgres \
  golang:1.25-alpine go run ./cmd/seed -table=patient
```

#### Option B: Running via Host Go CLI
```bash
# Seed all tables:
go run ./cmd/seed

# Seed specific table:
go run ./cmd/seed -table=hospital
go run ./cmd/seed -table=patient

# Bypass production safeguard (if needed):
go run ./cmd/seed -force
```

---

## Testing (TDD Suite)

Run the full Go test suite (unit tests and database integration tests):

```bash
go test -v ./...
```

> **Tip:** If running outside standard system paths, ensure the Go binary directory is in your `PATH` (e.g., `export PATH=$PATH:/usr/local/go/bin:~/go/bin`).

---

## Environment Variables

| Variable | Default | Description |
| :--- | :--- | :--- |
| `PORT` | `8080` | Port for the Go HTTP server |
| `GIN_MODE` | `debug` | Gin framework mode (`debug` or `release`; set to `release` in Docker Compose) |
| `NGINX_PORT` | `80` | Host port for Nginx HTTP (redirects to HTTPS via 308) |
| `NGINX_SSL_PORT` | `443` | Host port for Nginx HTTPS (SSL/TLS) |
| `DB_HOST` | `localhost` | PostgreSQL host (`postgres` when inside Docker) |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `admin` | PostgreSQL username |
| `DB_PASSWORD` | `postgres` | PostgreSQL password |
| `DB_NAME` | `hospital_middleware_db` | PostgreSQL database name |
| `DB_SSLMODE` | `disable` | PostgreSQL SSL mode (`disable`, `require`, etc.) |
| `DB_AUTO_MIGRATE` | `true` | Automatically run migrations on startup (`true` / `false`) |
| `DB_MAX_OPEN_CONNS` | `25` | Maximum number of open connections in PostgreSQL connection pool |
| `DB_MAX_IDLE_CONNS` | `25` | Maximum number of idle connections in PostgreSQL connection pool |
| `DB_CONN_MAX_LIFETIME_MINUTES` | `5` | Connection maximum lifetime in minutes before recycling |
| `DB_CONN_MAX_IDLE_TIME_MINUTES` | `5` | Maximum idle connection duration in minutes |
| `JWT_SECRET` | `hospital-middleware-super-secret-key-32bytes` | Secret key used for signing and validating JWT session tokens (minimum 32 characters) |
| `JWT_EXPIRY_SECONDS` | `86400` | Expiration time for JWT session tokens in seconds (default: 24h) |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:3000,http://localhost:5173` | Comma-separated list of allowed origins for CORS (supports `*` for all) |
| `CORS_MAX_AGE_SECONDS` | `86400` | Browser cache duration for CORS preflight OPTIONS in seconds (default: 24h) |

---

## Troubleshooting & FAQ

### 1. Nginx fails to start: missing `server.crt` or `server.key`
**Symptoms:** Nginx container exits with status code 1; log says: `cannot load certificate "/etc/nginx/ssl/server.crt": BIO_new_file() failed`.  
**Solution:** SSL certificate files are git-ignored for security. Generate them once by running:
```bash
chmod +x ./nginx/ssl/generate-cert.sh
./nginx/ssl/generate-cert.sh
docker compose up -d --force-recreate nginx
```

### 2. Port conflict: `Bind for 0.0.0.0:80 failed: port is already allocated`
**Symptoms:** Docker Compose fails to start `nginx` or `postgres` because ports `80`, `443`, `8080`, or `5432` are in use by local services (Apache, Nginx, local PostgreSQL).  
**Solution:** Change the exposed host ports in your `.env` file without changing internal container wiring:
```env
NGINX_PORT=8088
NGINX_SSL_PORT=8443
PORT=8081
DB_PORT=5433
```
Then restart: `docker compose up -d`.

### 3. Database password mismatch: `failed SASL auth: FATAL: password authentication failed`
**Symptoms:** The Go API cannot connect to PostgreSQL on startup or during seeding.  
**Cause:** Docker named volume `postgres_data` preserves the database credentials set when PostgreSQL was first initialized. If you changed `DB_PASSWORD` in `.env` after the container was already created, the database still uses the old password.  
**Solution:** Reset the volume and recreate the database with your new `.env` settings:
```bash
# WARNING: This deletes existing local database data
docker compose down -v
docker compose up -d --build
# Re-run seeder:
docker run --rm --network bridge-network -v "$PWD":/app -w /app --env-file .env -e DB_HOST=postgres golang:1.25-alpine go run ./cmd/seed
```

### 4. `POST /staff/create` returns `400 Bad Request: hospital not found`
**Symptoms:** Registering a new staff member fails with `{"detail": "hospital not found"}`.  
**Cause:** Database migrations create the tables, but the `hospital` table has no records until the seeder is executed.  
**Solution:** Run the seeder to populate default hospital codes (`HOSP001` - `HOSP010`):
```bash
docker run --rm --network bridge-network -v "$PWD":/app -w /app --env-file .env -e DB_HOST=postgres golang:1.25-alpine go run ./cmd/seed
```

### 5. `POST /staff/create` returns validation error on `Username`
**Symptoms:** `Key: 'CreateStaffRequest.Username' Error:Field validation for 'Username' failed on the 'alphanum' tag`.  
**Solution:** Usernames must be strictly alphanumeric English characters (`[a-zA-Z0-9]`). Underscores (`_`), dashes (`-`), spaces, and Thai characters are rejected. Use `somchai` or `nurseann` instead of `nurse_ann`.

### 6. `POST /staff/create` returns validation error on `Password`
**Symptoms:** `Field validation for 'Password' failed on the 'password' tag`.  
**Solution:** Check that the password satisfies all security criteria:
- Length: 8 to 32 characters
- English characters and numbers only (no Thai characters or spaces)
- At least 1 lowercase letter (`a-z`)
- At least 1 uppercase letter (`A-Z`)
- At least 1 digit (`0-9`)
- At least 1 special character (`!@#$%^&*()-_=+[]{}|;:'",.<>/?`~\\`)

### 7. Startup fatal error: `jwt secret must be at least 32 characters long`
**Symptoms:** API container crashes on startup with `[FATAL] Invalid configuration: jwt secret must be at least 32 characters long`.  
**Solution:** For cryptographic security, `JWT_SECRET` must contain 32 or more characters. Ensure your `.env` has:
```env
JWT_SECRET=your_jwt_secret_key_at_least_32_characters_long
```

### 8. SSL certificate warning in browser or curl (`certificate verify failed`)
**Symptoms:** Browser warns "Your connection is not private", or `curl` fails with SSL handshake error.  
**Solution:** Because the certificate is self-signed for local development:
- **curl**: Pass the `-k` (or `--insecure`) flag: `curl -k https://localhost/health`
- **Chrome / Edge**: Click "Advanced" -> "Proceed to localhost (unsafe)".
- **Firefox**: Click "Advanced" -> "Accept the Risk and Continue".

### 9. Windows script error: `\r: command not found` in `generate-cert.sh`
**Symptoms:** Running `./nginx/ssl/generate-cert.sh` on Windows fails with carriage return syntax errors.  
**Solution:** Convert CRLF to LF line endings using Git Bash or dos2unix:
```bash
dos2unix ./nginx/ssl/generate-cert.sh
# Or run with sh directly:
sh ./nginx/ssl/generate-cert.sh
```
