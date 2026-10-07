# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install certificates and git
RUN apk add --no-cache ca-certificates git tzdata

# Leverage dependency caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/api ./cmd/api

# Production stage
FROM alpine:3.21

WORKDIR /app

# Install ca-certificates and tzdata for TLS and timezone support
RUN apk --no-cache add ca-certificates tzdata

# Run as non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# Copy executable from builder
COPY --from=builder /app/bin/api /app/api

EXPOSE 8080

CMD ["/app/api"]
