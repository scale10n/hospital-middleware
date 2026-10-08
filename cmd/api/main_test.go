package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hospital-middleware/internal/config"
	"hospital-middleware/internal/database"
	"hospital-middleware/internal/middleware"
	"hospital-middleware/internal/response"
	"hospital-middleware/internal/service"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestHealthRoute_NilDB(t *testing.T) {
	router := setupRouter(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503 when db is nil, got %d", w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if response["database"] != "disconnected" {
		t.Errorf("expected database 'disconnected', got %q", response["database"])
	}
}

func TestStaffCreateRoute_NilDB(t *testing.T) {
	router := setupRouter(nil)

	payload := `{"username":"tester","password":"Password123!","hospital":"HOSP001"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/staff/create", bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500 when db is nil, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response middleware.ProblemDetails
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if response.Detail != "failed to query hospital" {
		t.Errorf("expected detail 'failed to query hospital', got %q", response.Detail)
	}
	if response.Status != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", response.Status)
	}
}

func TestStaffCreateRoute_RealDB(t *testing.T) {
	cfg := config.Load()
	db, err := database.New(cfg.DB)
	if err != nil {
		t.Skipf("skipping test: database connection not available: %v", err)
	}
	defer db.Close()

	// Clean up tester username before and after test for idempotency
	_, _ = db.Exec("DELETE FROM staff WHERE username = $1", "tester")
	defer func() {
		_, _ = db.Exec("DELETE FROM staff WHERE username = $1", "tester")
	}()

	router := setupRouter(db)

	// 1. Success case: Valid hospital HN (seeded in database)
	payload := `{"username":"tester","password":"Password123!","hospital":"HOSP001"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/staff/create", bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp response.Response[service.StaffResponseData]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("expected status 'success', got %q", resp.Status)
	}
	if resp.Data.Staff.Username != "tester" {
		t.Errorf("expected username 'tester', got %q", resp.Data.Staff.Username)
	}
	if resp.Data.Staff.HospitalHN != "HOSP001" {
		t.Errorf("expected hospital_hn 'HOSP001', got %q", resp.Data.Staff.HospitalHN)
	}
	if resp.Data.Staff.HospitalName == "" {
		t.Errorf("expected non-empty hospital_name")
	}
	if resp.Data.Staff.ID == "" {
		t.Errorf("expected non-empty staff ID")
	}
	if resp.Data.Session.ExpiresAt.IsZero() {
		t.Errorf("expected non-zero session expires_at")
	}

	// Verify that session_token HttpOnly cookie was set
	cookieHeader := w.Header().Get("Set-Cookie")
	if !strings.Contains(cookieHeader, "session_token=") {
		t.Errorf("expected Set-Cookie to contain session_token, got %q", cookieHeader)
	}
	if !strings.Contains(cookieHeader, "HttpOnly") {
		t.Errorf("expected Set-Cookie to have HttpOnly flag, got %q", cookieHeader)
	}

	// Verify that staff record exists in database table staff
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM staff WHERE username = $1", "tester").Scan(&count); err != nil || count != 1 {
		t.Errorf("expected 1 staff record in table staff, got %d (err: %v)", count, err)
	}

	// Verify that session record exists in database table staff_session
	var sessionCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM staff_session WHERE staff_id = (SELECT id FROM staff WHERE username = $1)", "tester").Scan(&sessionCount); err != nil || sessionCount != 1 {
		t.Errorf("expected 1 session record in table staff_session, got %d (err: %v)", sessionCount, err)
	}

	// 2. Failure case: Unknown hospital HN (does not exist in table hospital)
	invalidPayload := `{"username":"tester","password":"Password123!","hospital":"NON_EXISTENT_HN"}`
	wBad := httptest.NewRecorder()
	reqBad, _ := http.NewRequest(http.MethodPost, "/staff/create", bytes.NewReader([]byte(invalidPayload)))
	reqBad.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wBad, reqBad)

	if wBad.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for non-existent hospital, got %d. Body: %s", wBad.Code, wBad.Body.String())
	}

	var errResp middleware.ProblemDetails
	if err := json.Unmarshal(wBad.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}
	if errResp.Detail != "hospital not found" {
		t.Errorf("expected detail 'hospital not found', got %q", errResp.Detail)
	}
	if errResp.Status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", errResp.Status)
	}
}
