# Hospital Middleware

A Go-based API middleware service built with [Gin Web Framework](https://github.com/gin-gonic/gin).

---

## Table of Contents

- [Features](#features)
- [Project Structure](#project-structure)
- [Prerequisites](#prerequisites)
- [Getting Started](#getting-started)
  - [Run Locally](#run-locally)
  - [Run with Docker Compose](#run-with-docker-compose)
- [API Endpoints](#api-endpoints)
- [Environment Variables](#environment-variables)
- [Troubleshooting](#troubleshooting)

---

## Features

- **Gin HTTP Server**: Fast and lightweight REST API framework.
- **Dockerized**: Multi-stage Dockerfile producing a minimal, secure non-root Alpine container.
- **Docker Compose**: Pre-configured with automatic health checks and port forwarding.
- **Go Modules**: Dependency management configured for Go 1.25+.

---

## Project Structure

```text
hospital-middleware/
├── cmd/
│   └── api/
│       └── main.go          # Application entrypoint
├── Dockerfile               # Multi-stage Docker build
├── docker-compose.yml       # Docker Compose service definition
├── .dockerignore            # Excluded build context files
├── go.mod                   # Go module definitions
├── go.sum                   # Dependency checksums
└── README.md
```

---

## Prerequisites

- **Go**: `1.25.0` or higher
- **Docker**: `20.10+` (or Docker Desktop)
- **Docker Compose**: `v2+` / `v5+`

---

## Getting Started

### Run Locally

1. **Install dependencies**:
   ```bash
   go mod download
   ```

2. **Run the server**:
   ```bash
   go run ./cmd/api
   ```
   *Or build the binary:*
   ```bash
   go build -o bin/api ./cmd/api
   ./bin/api
   ```

3. **Verify the server**:
   ```bash
   curl http://localhost:8080/ping
   ```

---

### Run with Docker Compose

#### 1. Build and start the container
```bash
docker compose up -d --build
```

#### 2. Check container status
```bash
docker compose ps
```

#### 3. View live logs
```bash
docker compose logs -f
```

#### 4. Test the API
```bash
curl http://localhost:8080/ping
```

#### 5. Stop the container
```bash
docker compose down
```

---

## API Endpoints

| Method | Endpoint | Description | Response Example |
| :--- | :--- | :--- | :--- |
| `GET` | `/ping` | Health/Ping check | `{"message":"pong"}` |

### Example Request

```bash
curl -i http://localhost:8080/ping
```

**Response:**
```http
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8

{"message":"pong"}
```

---

## Environment Variables

| Variable | Default | Description |
| :--- | :--- | :--- |
| `PORT` | `8080` | Port on which the HTTP server listens |
| `GIN_MODE` | `debug` | Gin runtime mode (`debug` or `release`) |

You can override these in `docker-compose.yml` or via the command line:

```bash
PORT=9000 GIN_MODE=release go run ./cmd/api
```

---

## Troubleshooting

### Dependency version conflicts (`requires go >= 1.26`)
Avoid using `go get -u` as it forces all transitive packages to update to their bleeding-edge versions which may require a newer Go compiler than installed. Instead, add dependencies directly:
```bash
go get <package-name>
go mod tidy
```
