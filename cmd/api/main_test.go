package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestStaffLoginRoute_NilDB(t *testing.T) {
	router := setupRouter(nil)

	payload := `{"username":"tester","password":"Password123!","hospital":"HOSP001"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/staff/login", bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500 when db is nil, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response middleware.ProblemDetails
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if response.Status != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", response.Status)
	}
}

func TestStaffLoginRoute_RealDB(t *testing.T) {
	cfg := config.Load()
	db, err := database.New(cfg.DB)
	if err != nil {
		t.Skipf("skipping test: database connection not available: %v", err)
	}
	defer db.Close()

	// Clean up tester username before and after test for idempotency
	_, _ = db.Exec("DELETE FROM staff WHERE username = $1", "logintester")
	defer func() {
		_, _ = db.Exec("DELETE FROM staff WHERE username = $1", "logintester")
	}()

	router := setupRouter(db)

	// Create a staff user first
	createPayload := `{"username":"logintester","password":"Password123!","hospital":"HOSP001"}`
	wCreate := httptest.NewRecorder()
	reqCreate, _ := http.NewRequest(http.MethodPost, "/staff/create", bytes.NewReader([]byte(createPayload)))
	reqCreate.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wCreate, reqCreate)

	if wCreate.Code != http.StatusCreated {
		t.Fatalf("failed to seed staff for login test: %d, body: %s", wCreate.Code, wCreate.Body.String())
	}

	// 1. Success case: valid login (Measure response time < 200ms)
	loginPayload := `{"username":"logintester","password":"Password123!","hospital":"HOSP001"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/staff/login", bytes.NewReader([]byte(loginPayload)))
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	router.ServeHTTP(w, req)
	elapsed := time.Since(start)

	t.Logf("POST /staff/login response time: %v", elapsed)
	if elapsed >= 200*time.Millisecond {
		t.Errorf("expected response time < 200ms, got %v", elapsed)
	}

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp response.Response[service.StaffResponseData]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("expected status 'success', got %q", resp.Status)
	}
	if resp.Message != "login successful" {
		t.Errorf("expected message 'login successful', got %q", resp.Message)
	}
	if resp.Data.Staff.Username != "logintester" {
		t.Errorf("expected username 'logintester', got %q", resp.Data.Staff.Username)
	}
	if resp.Data.Staff.HospitalHN != "HOSP001" {
		t.Errorf("expected hospital 'HOSP001', got %q", resp.Data.Staff.HospitalHN)
	}

	// Verify Set-Cookie
	cookieHeader := w.Header().Get("Set-Cookie")
	if !strings.Contains(cookieHeader, "session_token=") {
		t.Errorf("expected Set-Cookie to contain session_token, got %q", cookieHeader)
	}

	// 2. Failure case: wrong password
	badPwPayload := `{"username":"logintester","password":"WrongPassword123!","hospital":"HOSP001"}`
	wBadPw := httptest.NewRecorder()
	reqBadPw, _ := http.NewRequest(http.MethodPost, "/staff/login", bytes.NewReader([]byte(badPwPayload)))
	reqBadPw.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wBadPw, reqBadPw)

	if wBadPw.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d. Body: %s", wBadPw.Code, wBadPw.Body.String())
	}

	// 3. Failure case: unknown username
	unknownPayload := `{"username":"unknownuser123","password":"Password123!","hospital":"HOSP001"}`
	wUnknown := httptest.NewRecorder()
	reqUnknown, _ := http.NewRequest(http.MethodPost, "/staff/login", bytes.NewReader([]byte(unknownPayload)))
	reqUnknown.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wUnknown, reqUnknown)

	if wUnknown.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d. Body: %s", wUnknown.Code, wUnknown.Body.String())
	}

	// 4. Failure case: unknown hospital HN
	unknownHospPayload := `{"username":"logintester","password":"Password123!","hospital":"UNKNOWN_HN"}`
	wUnknownHosp := httptest.NewRecorder()
	reqUnknownHosp, _ := http.NewRequest(http.MethodPost, "/staff/login", bytes.NewReader([]byte(unknownHospPayload)))
	reqUnknownHosp.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wUnknownHosp, reqUnknownHosp)

	if wUnknownHosp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 when hospital does not exist, got %d. Body: %s", wUnknownHosp.Code, wUnknownHosp.Body.String())
	}

	var unknownHospResp middleware.ProblemDetails
	if err := json.Unmarshal(wUnknownHosp.Body.Bytes(), &unknownHospResp); err != nil {
		t.Fatalf("failed to parse problem details: %v", err)
	}
	if unknownHospResp.Detail != "hospital not found" {
		t.Errorf("expected detail 'hospital not found', got %q", unknownHospResp.Detail)
	}

	// 5. Failure case: missing hospital
	missingHospPayload := `{"username":"logintester","password":"Password123!"}`
	wMissingHosp := httptest.NewRecorder()
	reqMissingHosp, _ := http.NewRequest(http.MethodPost, "/staff/login", bytes.NewReader([]byte(missingHospPayload)))
	reqMissingHosp.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wMissingHosp, reqMissingHosp)

	if wMissingHosp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 when hospital is missing, got %d. Body: %s", wMissingHosp.Code, wMissingHosp.Body.String())
	}

	// 6. Security verification: SQL Injection payload in hospital parameter
	sqlInjPayload := `{"username":"logintester","password":"Password123!","hospital":"' OR '1'='1; --"}`
	wInj := httptest.NewRecorder()
	reqInj, _ := http.NewRequest(http.MethodPost, "/staff/login", bytes.NewReader([]byte(sqlInjPayload)))
	reqInj.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wInj, reqInj)

	if wInj.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for SQL injection attempt, got %d. Body: %s", wInj.Code, wInj.Body.String())
	}
	var sqlInjResp middleware.ProblemDetails
	if err := json.Unmarshal(wInj.Body.Bytes(), &sqlInjResp); err != nil {
		t.Fatalf("failed to parse problem details: %v", err)
	}
	if sqlInjResp.Detail != "hospital not found" {
		t.Errorf("expected detail 'hospital not found', got %q", sqlInjResp.Detail)
	}

	// 7. Performance verification: Response time < 200ms for typical queries
	const iterations = 10
	var totalDuration time.Duration
	for i := 0; i < iterations; i++ {
		wPerf := httptest.NewRecorder()
		reqPerf, _ := http.NewRequest(http.MethodPost, "/staff/login", bytes.NewReader([]byte(loginPayload)))
		reqPerf.Header.Set("Content-Type", "application/json")

		iterStart := time.Now()
		router.ServeHTTP(wPerf, reqPerf)
		iterElapsed := time.Since(iterStart)

		totalDuration += iterElapsed
		if iterElapsed >= 200*time.Millisecond {
			t.Errorf("iteration %d: expected response time < 200ms, got %v", i+1, iterElapsed)
		}
		if wPerf.Code != http.StatusOK {
			t.Errorf("iteration %d: expected status 200, got %d", i+1, wPerf.Code)
		}
	}
	avgDuration := totalDuration / iterations
	t.Logf("POST /staff/login average response time over %d iterations: %v (target: < 200ms)", iterations, avgDuration)
}
