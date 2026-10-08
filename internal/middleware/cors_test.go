package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"hospital-middleware/internal/config"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestCORS_Preflight_WithoutOrigin(t *testing.T) {
	r := gin.New()
	r.Use(CORS(config.CORSConfig{
		AllowedOrigins: []string{"http://localhost:3000"},
		MaxAgeSeconds:  86400,
	}))
	r.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodOptions, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d for OPTIONS preflight, got %d", http.StatusNoContent, w.Code)
	}

	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin '*', got %q", origin)
	}
	if maxAge := w.Header().Get("Access-Control-Max-Age"); maxAge != "86400" {
		t.Errorf("expected Access-Control-Max-Age '86400', got %q", maxAge)
	}
}

func TestCORS_Preflight_WithAllowedOrigin(t *testing.T) {
	r := gin.New()
	r.Use(CORS(config.CORSConfig{
		AllowedOrigins: []string{"http://localhost:3000", "http://localhost:5173"},
		MaxAgeSeconds:  86400,
	}))
	r.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	clientOrigin := "http://localhost:3000"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", clientOrigin)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d for OPTIONS preflight, got %d", http.StatusNoContent, w.Code)
	}

	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != clientOrigin {
		t.Errorf("expected Access-Control-Allow-Origin %q, got %q", clientOrigin, origin)
	}
	if creds := w.Header().Get("Access-Control-Allow-Credentials"); creds != "true" {
		t.Errorf("expected Access-Control-Allow-Credentials 'true', got %q", creds)
	}
	if maxAge := w.Header().Get("Access-Control-Max-Age"); maxAge != "86400" {
		t.Errorf("expected Access-Control-Max-Age '86400', got %q", maxAge)
	}
	if vary := w.Header().Get("Vary"); vary != "Origin" {
		t.Errorf("expected Vary 'Origin', got %q", vary)
	}
}

func TestCORS_NormalRequest_WithoutOrigin(t *testing.T) {
	r := gin.New()
	r.Use(CORS())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin '*', got %q", origin)
	}
}

func TestCORS_NormalRequest_WithAllowedOrigin(t *testing.T) {
	r := gin.New()
	r.Use(CORS(config.CORSConfig{
		AllowedOrigins: []string{"https://hospital.example.com"},
		MaxAgeSeconds:  3600,
	}))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	clientOrigin := "https://hospital.example.com"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", clientOrigin)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != clientOrigin {
		t.Errorf("expected Access-Control-Allow-Origin %q, got %q", clientOrigin, origin)
	}
	if creds := w.Header().Get("Access-Control-Allow-Credentials"); creds != "true" {
		t.Errorf("expected Access-Control-Allow-Credentials 'true', got %q", creds)
	}
	if maxAge := w.Header().Get("Access-Control-Max-Age"); maxAge != "3600" {
		t.Errorf("expected Access-Control-Max-Age '3600', got %q", maxAge)
	}
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	r := gin.New()
	r.Use(CORS(config.CORSConfig{
		AllowedOrigins: []string{"https://trusted.hospital.com"},
		MaxAgeSeconds:  86400,
	}))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	evilOrigin := "https://malicious-website.com"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", evilOrigin)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// For unauthorized origins, Access-Control-Allow-Origin must NOT be set to the malicious origin
	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin == evilOrigin {
		t.Errorf("expected evil origin %q to NOT be allowed, but it was set", evilOrigin)
	}
}
