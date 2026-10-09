package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hospital-middleware/internal/auth"
	"hospital-middleware/internal/middleware"

	"github.com/gin-gonic/gin"
)

func TestCookieAuth_MissingCookie(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret-key-at-least-32-bytes-long", time.Hour)

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.CookieAuth(jwtService))
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}

	var problem middleware.ProblemDetails
	if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
		t.Fatalf("failed to parse problem details: %v", err)
	}

	if problem.Status != http.StatusUnauthorized {
		t.Errorf("expected problem status 401, got %d", problem.Status)
	}
	if problem.Detail != middleware.ErrSessionCookieMissing.Error() {
		t.Errorf("expected detail %q, got %q", middleware.ErrSessionCookieMissing.Error(), problem.Detail)
	}
}

func TestCookieAuth_EmptyCookie(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret-key-at-least-32-bytes-long", time.Hour)

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.CookieAuth(jwtService))
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  middleware.SessionCookieName,
		Value: "   ",
	})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestCookieAuth_InvalidToken(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret-key-at-least-32-bytes-long", time.Hour)

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.CookieAuth(jwtService))
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  middleware.SessionCookieName,
		Value: "invalid.jwt.token",
	})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestCookieAuth_ExpiredToken(t *testing.T) {
	expiredJWTService := auth.NewJWTService("test-secret-key-at-least-32-bytes-long", -time.Hour)
	token, err := expiredJWTService.GenerateToken("staff-1", "hosp-1", "nurse_alice")
	if err != nil {
		t.Fatalf("failed to generate expired token: %v", err)
	}

	validService := auth.NewJWTService("test-secret-key-at-least-32-bytes-long", time.Hour)

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.CookieAuth(validService))
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  middleware.SessionCookieName,
		Value: token,
	})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestCookieAuth_MissingHospitalClaim(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret-key-at-least-32-bytes-long", time.Hour)
	token, err := jwtService.GenerateToken("staff-1", "", "nurse_alice")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.CookieAuth(jwtService))
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  middleware.SessionCookieName,
		Value: token,
	})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestCookieAuth_MissingStaffIDClaim(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret-key-at-least-32-bytes-long", time.Hour)
	token, err := jwtService.GenerateToken("", "hosp-1", "nurse_alice")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.CookieAuth(jwtService))
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  middleware.SessionCookieName,
		Value: token,
	})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestCookieAuth_ValidToken(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret-key-at-least-32-bytes-long", time.Hour)
	token, err := jwtService.GenerateToken("staff-123", "hosp-456", "dr_bob")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	var capturedHospitalID string
	var capturedStaffID string
	var capturedUsername string

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.CookieAuth(jwtService))
	r.GET("/protected", func(c *gin.Context) {
		capturedHospitalID, _ = middleware.GetHospitalID(c)
		capturedStaffID, _ = middleware.GetStaffID(c)
		claims, ok := middleware.GetStaffClaims(c)
		if ok && claims != nil {
			capturedUsername = claims.Username
		}
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  middleware.SessionCookieName,
		Value: token,
	})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	if capturedHospitalID != "hosp-456" {
		t.Errorf("expected hospital ID 'hosp-456', got %q", capturedHospitalID)
	}
	if capturedStaffID != "staff-123" {
		t.Errorf("expected staff ID 'staff-123', got %q", capturedStaffID)
	}
	if capturedUsername != "dr_bob" {
		t.Errorf("expected username 'dr_bob', got %q", capturedUsername)
	}
}
