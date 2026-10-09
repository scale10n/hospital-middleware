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
  - [Endpoints](#endpoints)
  - [Interactive API Documentation (Swagger UI)](#interactive-api-documentation-swagger-ui)
  - [Request Validation Rules](#request-validation-rules)
  - [Testing via curl](#testing-via-curl)
  - [Error Handling (RFC 7807 Problem Details)](#error-handling-rfc-7807-problem-details)
- [Database Migrations](#database-migrations)
  - [1. Automatic Migrations on Startup](#1-automatic-migrations-on-startup)
  - [2. Migration Files Overview](#2-migration-files-overview)
  - [3. Manual CLI Migrations](#3-manual-cli-migrations)
- [Database Seeder (Mock Data)](#database-seeder-mock-data)
- [Testing (TDD)](#testing-tdd)
- [Environment Variables](#environment-variables)
- [Troubleshooting](#troubleshooting)

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

- **Docker**: `20.10+` (or Docker Desktop)
- **Docker Compose**: `v2+`
- **Go**: `1.25.0` or higher *(only if developing/running outside Docker)*
- **swag CLI**: *(optional/recommended, for generating Swagger documentation)*: `go install github.com/swaggo/swag/cmd/swag@latest`
- **OpenSSL**: *(used to generate local SSL certificates; Docker fallback is automatic)*
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

# 5. Verify health via HTTPS
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

Because private keys (`*.key`) and certificates (`*.crt`) are excluded from Git for security, generate them once locally:

```bash
chmod +x ./nginx/ssl/generate-cert.sh
./nginx/ssl/generate-cert.sh
```

Optional arguments:
```bash
# Custom domain and validity period (days)
./nginx/ssl/generate-cert.sh localhost 365
```

This generates `server.crt` and `server.key` inside `nginx/ssl/` with Subject Alternative Names (SAN) supporting `localhost`, `127.0.0.1`, and `::1`. *(If OpenSSL is not installed on the host, the script automatically falls back to Docker and ensures file permissions are set correctly to `600` for the key and `644` for the certificate).*

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

| Method | Endpoint | Description | Status Code | Content-Type |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/swagger/*any` | Interactive Swagger UI API documentation | `200 OK` | `text/html` |
| `GET` | `/health` | Readiness probe & PostgreSQL DB check | `200 OK` | `application/json` |
| `POST` | `/staff/create` | Validate hospital HN, create staff member, hash password, persist session & issue JWT | `201 Created` | `application/json` |
| `POST` | `/staff/login` | Authenticate staff credentials, persist session & issue JWT | `200 OK` | `application/json` |
| `POST` | `/patient/search` | Search patients within staff's hospital matching all provided criteria (CookieAuth) | `200 OK` | `application/json` |

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

---

### Testing via curl

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
      "hospital_name": "Bangkok General Hospital",
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
      "hospital_name": "Bangkok General Hospital",
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

#### 2. Via HTTP Auto-Redirect (Port 80 -> 443 via 308)
Nginx issues an **HTTP 308 Permanent Redirect**, which guarantees that clients preserve the HTTP method (`POST`) and request body when redirecting to HTTPS:
```bash
# Inspect 308 Permanent Redirect response
curl -i http://localhost/health

# Automatically follow redirect (-L) for POST request
curl -k -L -i -X POST http://localhost/staff/create \
  -H "Content-Type: application/json" \
  -d '{
    "username": "nurse_ann",
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

# Seed only the patient table (for /patient/search API testing)
go run ./cmd/seed -table=patient
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

## Troubleshooting

### 1. Nginx fails to start: missing `server.crt` or `server.key`
SSL certificate files are git-ignored for security. Generate them by running:
```bash
./nginx/ssl/generate-cert.sh
docker compose up -d --force-recreate nginx
```

### 2. Startup fatal error: `jwt secret must be at least 32 characters long`
For cryptographic security, `JWT_SECRET` is validated on startup and requires at least 32 characters. Ensure your `.env` contains a sufficiently long secret key:
```env
JWT_SECRET=your_jwt_secret_key_at_least_32_characters_long
```

### 3. SSL certificate warning in browser or curl (`certificate verify failed`)
Because the certificate is self-signed for local development:
- **curl**: Pass the `-k` (or `--insecure`) flag.
- **Chrome / Firefox**: Click "Advanced" -> "Proceed to localhost (unsafe)".

### 4. POST /staff/create returns validation error on password
Check that the password satisfies all criteria:
- Minimum 8 characters, maximum 32 characters
- English characters and numbers only (no Thai characters or spaces)
- At least 1 lowercase letter, 1 uppercase letter, 1 digit, and 1 special character (`!@#$%^&*...`)

### 5. Dependency version conflicts (`requires go >= 1.26`)
Avoid running `go get -u` across all packages, as it upgrades transitive dependencies to unreleased Go versions. Instead, install specific packages:
```bash
go get <package-name>
go mod tidy
```
