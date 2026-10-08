package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockPinger implements DBPinger for testing without a real database
type mockPinger struct {
	err error
}

func (m *mockPinger) PingContext(ctx context.Context) error {
	return m.err
}


func TestHealthHandler_HealthCheck_NilDB(t *testing.T) {
	handler := NewHealthHandler(nil)
	r := gin.New()
	r.GET("/health", handler.HealthCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if resp["database"] != "disconnected" {
		t.Errorf("expected 'disconnected', got %q", resp["database"])
	}
}

func TestHealthHandler_HealthCheck_PingError(t *testing.T) {
	mock := &mockPinger{err: errors.New("connection refused")}
	handler := NewHealthHandler(mock)
	r := gin.New()
	r.GET("/health", handler.HealthCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if resp["error"] != "connection refused" {
		t.Errorf("expected 'connection refused', got %q", resp["error"])
	}
}

func TestHealthHandler_HealthCheck_Success(t *testing.T) {
	mock := &mockPinger{err: nil}
	handler := NewHealthHandler(mock)
	r := gin.New()
	r.GET("/health", handler.HealthCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if resp["status"] != "healthy" || resp["database"] != "connected" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestRegisterRoutes(t *testing.T) {
	mock := &mockPinger{err: nil}
	handler := NewHealthHandler(mock)
	r := gin.New()
	RegisterRoutes(r, handler)

	// Test root health
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected /health status 200, got %d", w.Code)
	}
}
