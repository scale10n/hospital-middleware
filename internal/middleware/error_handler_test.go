package middleware_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"hospital-middleware/internal/middleware"
	"hospital-middleware/internal/service"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestErrorHandler_NoError(t *testing.T) {
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.GET("/success", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/success", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestErrorHandler_RFC7807_HospitalNotFound(t *testing.T) {
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.GET("/hospital-error", func(c *gin.Context) {
		_ = c.Error(service.ErrHospitalNotFound)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/hospital-error", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	// Verify RFC 7807 Content-Type header
	contentType := w.Header().Get("Content-Type")
	if contentType != middleware.ContentTypeProblemJSON {
		t.Errorf("expected Content-Type %q, got %q", middleware.ContentTypeProblemJSON, contentType)
	}

	var problem middleware.ProblemDetails
	if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
		t.Fatalf("failed to unmarshal ProblemDetails JSON: %v", err)
	}

	if problem.Status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", problem.Status)
	}
	if problem.Title != "Bad Request" {
		t.Errorf("expected title 'Bad Request', got %q", problem.Title)
	}
	if problem.Detail != "hospital not found" {
		t.Errorf("expected detail 'hospital not found', got %q", problem.Detail)
	}
	if problem.Type != "/errors/hospital-not-found" {
		t.Errorf("expected type '/errors/hospital-not-found', got %q", problem.Type)
	}
	if problem.Instance != "/hospital-error" {
		t.Errorf("expected instance '/hospital-error', got %q", problem.Instance)
	}
}

func TestErrorHandler_RFC7807_StaffAlreadyExists(t *testing.T) {
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.GET("/staff-duplicate", func(c *gin.Context) {
		_ = c.Error(service.ErrStaffAlreadyExists)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/staff-duplicate", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", w.Code)
	}

	var problem middleware.ProblemDetails
	if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
		t.Fatalf("failed to unmarshal ProblemDetails JSON: %v", err)
	}

	if problem.Status != http.StatusConflict {
		t.Errorf("expected status 409, got %d", problem.Status)
	}
	if problem.Detail != "staff already exists" {
		t.Errorf("expected detail 'staff already exists', got %q", problem.Detail)
	}
	if problem.Type != "/errors/staff-already-exists" {
		t.Errorf("expected type '/errors/staff-already-exists', got %q", problem.Type)
	}
}

func TestErrorHandler_RFC7807_InvalidCredentials(t *testing.T) {
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.GET("/login-error", func(c *gin.Context) {
		_ = c.Error(service.ErrInvalidCredentials)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/login-error", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != middleware.ContentTypeProblemJSON {
		t.Errorf("expected Content-Type %q, got %q", middleware.ContentTypeProblemJSON, contentType)
	}

	var problem middleware.ProblemDetails
	if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
		t.Fatalf("failed to unmarshal ProblemDetails JSON: %v", err)
	}

	if problem.Status != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", problem.Status)
	}
	if problem.Title != "Unauthorized" {
		t.Errorf("expected title 'Unauthorized', got %q", problem.Title)
	}
	if problem.Detail != "invalid username or password" {
		t.Errorf("expected detail 'invalid username or password', got %q", problem.Detail)
	}
	if problem.Type != "/errors/unauthorized" {
		t.Errorf("expected type '/errors/unauthorized', got %q", problem.Type)
	}
	if problem.Instance != "/login-error" {
		t.Errorf("expected instance '/login-error', got %q", problem.Instance)
	}
}

func TestErrorHandler_RFC7807_CustomAppError(t *testing.T) {
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.GET("/app-error", func(c *gin.Context) {
		_ = c.Error(middleware.NewAppError(http.StatusConflict, "duplicate record"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/app-error", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", w.Code)
	}

	var problem middleware.ProblemDetails
	_ = json.Unmarshal(w.Body.Bytes(), &problem)

	if problem.Status != http.StatusConflict {
		t.Errorf("expected status 409, got %d", problem.Status)
	}
	if problem.Title != "Conflict" {
		t.Errorf("expected title 'Conflict', got %q", problem.Title)
	}
	if problem.Detail != "duplicate record" {
		t.Errorf("expected detail 'duplicate record', got %q", problem.Detail)
	}
}

func TestErrorHandler_RFC7807_InternalError(t *testing.T) {
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.GET("/internal-error", func(c *gin.Context) {
		_ = c.Error(errors.New("db disk failure"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/internal-error", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}

	var problem middleware.ProblemDetails
	_ = json.Unmarshal(w.Body.Bytes(), &problem)

	if problem.Status != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", problem.Status)
	}
	if problem.Title != "Internal Server Error" {
		t.Errorf("expected title 'Internal Server Error', got %q", problem.Title)
	}
	if problem.Detail != "failed to query hospital" {
		t.Errorf("expected detail 'failed to query hospital', got %q", problem.Detail)
	}
}

func TestErrorHandler_RFC7807_PatientNotFound(t *testing.T) {
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.GET("/patient-not-found", func(c *gin.Context) {
		_ = c.Error(service.ErrPatientNotFound)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/patient-not-found", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}

	var problem middleware.ProblemDetails
	if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
		t.Fatalf("failed to parse problem details: %v", err)
	}

	if problem.Status != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", problem.Status)
	}
	if problem.Detail != "patient not found" {
		t.Errorf("expected detail 'patient not found', got %q", problem.Detail)
	}
	if problem.Type != "/errors/patient-not-found" {
		t.Errorf("expected type '/errors/patient-not-found', got %q", problem.Type)
	}
}

func TestErrorHandler_RFC7807_InvalidPatientID(t *testing.T) {
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.GET("/invalid-patient-id", func(c *gin.Context) {
		_ = c.Error(service.ErrInvalidPatientID)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/invalid-patient-id", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	var problem middleware.ProblemDetails
	if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
		t.Fatalf("failed to parse problem details: %v", err)
	}

	if problem.Status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", problem.Status)
	}
	if problem.Detail != service.ErrInvalidPatientID.Error() {
		t.Errorf("expected detail %q, got %q", service.ErrInvalidPatientID.Error(), problem.Detail)
	}
	if problem.Type != "/errors/validation-error" {
		t.Errorf("expected type '/errors/validation-error', got %q", problem.Type)
	}
}

