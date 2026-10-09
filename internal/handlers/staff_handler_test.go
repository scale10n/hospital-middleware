package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hospital-middleware/internal/middleware"
	"hospital-middleware/internal/response"
	"hospital-middleware/internal/service"

	"github.com/gin-gonic/gin"
)

// mockStaffService implements service.StaffService for handler testing.
type mockStaffService struct {
	output      *service.CreateStaffResult
	loginOutput *service.LoginStaffResult
	err         error
	loginErr    error
}

func (m *mockStaffService) CreateStaff(ctx context.Context, input service.CreateStaffInput) (*service.CreateStaffResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.output != nil {
		return m.output, nil
	}
	return &service.CreateStaffResult{
		Staff: &service.StaffResponseData{
			Staff: service.StaffInfoResponse{
				ID:           "mock-staff-id",
				Username:     input.Username,
				HospitalHN:   input.Hospital,
				HospitalName: "Mock Hospital",
				CreatedAt:    time.Now(),
			},
			Session: service.SessionInfoResponse{
				ExpiresAt: time.Now().Add(24 * time.Hour),
			},
		},
		Token: "mock-jwt-session-token",
	}, nil
}

func (m *mockStaffService) LoginStaff(ctx context.Context, input service.LoginStaffInput) (*service.LoginStaffResult, error) {
	if m.loginErr != nil {
		return nil, m.loginErr
	}
	if m.err != nil {
		return nil, m.err
	}
	if m.loginOutput != nil {
		return m.loginOutput, nil
	}
	hospHN := input.Hospital
	if hospHN == "" {
		hospHN = "HOSP001"
	}
	return &service.LoginStaffResult{
		Staff: &service.StaffResponseData{
			Staff: service.StaffInfoResponse{
				ID:           "mock-staff-id",
				Username:     input.Username,
				HospitalHN:   hospHN,
				HospitalName: "Mock Hospital",
				CreatedAt:    time.Now(),
			},
			Session: service.SessionInfoResponse{
				ExpiresAt: time.Now().Add(24 * time.Hour),
			},
		},
		Token: "mock-jwt-session-token",
	}, nil
}

func newTestStaffRouter(handler *StaffHandler) *gin.Engine {
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.POST("/staff/create", handler.CreateStaff)
	r.POST("/staff/login", handler.LoginStaff)
	return r
}

func TestStaffHandler_CreateStaff_Success(t *testing.T) {
	mockSvc := &mockStaffService{}
	handler := NewStaffHandlerWithService(mockSvc)
	r := newTestStaffRouter(handler)

	reqPayload := CreateStaffRequest{
		Username: "stafftestuser",
		Password: "SuperSecret123!",
		Hospital: "HOSP001",
	}
	body, err := json.Marshal(reqPayload)
	if err != nil {
		t.Fatalf("failed to marshal request payload: %v", err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/staff/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp response.Response[service.StaffResponseData]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("expected status 'success', got %q", resp.Status)
	}
	if resp.Message != "staff created successfully" {
		t.Errorf("expected message 'staff created successfully', got %q", resp.Message)
	}
	if resp.Data.Staff.Username != reqPayload.Username {
		t.Errorf("expected username %q, got %q", reqPayload.Username, resp.Data.Staff.Username)
	}
	if resp.Data.Staff.HospitalHN != reqPayload.Hospital {
		t.Errorf("expected hospital %q, got %q", reqPayload.Hospital, resp.Data.Staff.HospitalHN)
	}
	if resp.Data.Staff.HospitalName != "Mock Hospital" {
		t.Errorf("expected hospital_name 'Mock Hospital', got %q", resp.Data.Staff.HospitalName)
	}
	if resp.Data.Session.ExpiresAt.IsZero() {
		t.Errorf("expected non-zero session expires_at")
	}

	// Verify that session_token HttpOnly cookie was set
	cookieHeader := w.Header().Get("Set-Cookie")
	if !strings.Contains(cookieHeader, "session_token=mock-jwt-session-token") {
		t.Errorf("expected Set-Cookie to contain session_token, got %q", cookieHeader)
	}
	if !strings.Contains(cookieHeader, "HttpOnly") {
		t.Errorf("expected Set-Cookie to have HttpOnly flag, got %q", cookieHeader)
	}
}

func TestStaffHandler_CreateStaff_HospitalNotFound(t *testing.T) {
	mockSvc := &mockStaffService{err: service.ErrHospitalNotFound}
	handler := NewStaffHandlerWithService(mockSvc)
	r := newTestStaffRouter(handler)

	reqPayload := CreateStaffRequest{
		Username: "stafftestuser",
		Password: "SuperSecret123!",
		Hospital: "UNKNOWN_HN",
	}
	body, _ := json.Marshal(reqPayload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/staff/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 when hospital does not exist, got %d", w.Code)
	}

	var resp middleware.ProblemDetails
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	if resp.Detail != "hospital not found" {
		t.Errorf("expected detail %q, got %q", "hospital not found", resp.Detail)
	}
	if resp.Status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.Status)
	}
}

func TestStaffHandler_CreateStaff_StaffAlreadyExists(t *testing.T) {
	mockSvc := &mockStaffService{err: service.ErrStaffAlreadyExists}
	handler := NewStaffHandlerWithService(mockSvc)
	r := newTestStaffRouter(handler)

	reqPayload := CreateStaffRequest{
		Username: "stafftestuser",
		Password: "SuperSecret123!",
		Hospital: "HOSP001",
	}
	body, _ := json.Marshal(reqPayload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/staff/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409 when staff already exists, got %d", w.Code)
	}

	var resp middleware.ProblemDetails
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	if resp.Detail != "staff already exists" {
		t.Errorf("expected detail %q, got %q", "staff already exists", resp.Detail)
	}
	if resp.Status != http.StatusConflict {
		t.Errorf("expected status 409, got %d", resp.Status)
	}
}

func TestStaffHandler_CreateStaff_InternalError(t *testing.T) {
	mockSvc := &mockStaffService{err: errors.New("unexpected database error")}
	handler := NewStaffHandlerWithService(mockSvc)
	r := newTestStaffRouter(handler)

	reqPayload := CreateStaffRequest{
		Username: "stafftestuser",
		Password: "SuperSecret123!",
		Hospital: "HOSP001",
	}
	body, _ := json.Marshal(reqPayload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/staff/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}

	var resp middleware.ProblemDetails
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}
	if resp.Detail != "failed to query hospital" {
		t.Errorf("expected 'failed to query hospital', got %q", resp.Detail)
	}
	if resp.Status != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.Status)
	}
}

func TestStaffHandler_CreateStaff_InvalidJSON(t *testing.T) {
	mockSvc := &mockStaffService{}
	handler := NewStaffHandlerWithService(mockSvc)
	r := newTestStaffRouter(handler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/staff/create", bytes.NewReader([]byte("{invalid-json")))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid json, got %d", w.Code)
	}
}

func TestStaffHandler_CreateStaff_MissingFields(t *testing.T) {
	mockSvc := &mockStaffService{}
	handler := NewStaffHandlerWithService(mockSvc)
	r := newTestStaffRouter(handler)

	testCases := []struct {
		name    string
		payload map[string]string
	}{
		{
			name: "missing password and hospital",
			payload: map[string]string{
				"username": "user1",
			},
		},
		{
			name: "missing hospital",
			payload: map[string]string{
				"username": "user1",
				"password": "pwd",
			},
		},
		{
			name: "empty hospital",
			payload: map[string]string{
				"username": "user1",
				"password": "pwd",
				"hospital": "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.payload)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/staff/create", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status 400 for %s, got %d", tc.name, w.Code)
			}
		})
	}
}

func TestStaffHandler_CreateStaff_InvalidUsername(t *testing.T) {
	mockSvc := &mockStaffService{}
	handler := NewStaffHandlerWithService(mockSvc)
	r := newTestStaffRouter(handler)

	testCases := []struct {
		name     string
		username string
	}{
		{
			name:     "thai characters",
			username: "สมชาย123",
		},
		{
			name:     "with spaces",
			username: "user 123",
		},
		{
			name:     "with special symbols underscore",
			username: "user_name",
		},
		{
			name:     "with special symbols hyphen",
			username: "user-name",
		},
		{
			name:     "with special symbols at-sign",
			username: "user@hospital",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			payload := map[string]string{
				"username": tc.username,
				"password": "Password123!",
				"hospital": "HOSP001",
			}
			body, _ := json.Marshal(payload)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/staff/create", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status 400 for username %q (%s), got %d", tc.username, tc.name, w.Code)
			}
		})
	}
}

func TestStaffHandler_CreateStaff_ValidAlphanumericUsername(t *testing.T) {
	mockSvc := &mockStaffService{}
	handler := NewStaffHandlerWithService(mockSvc)
	r := newTestStaffRouter(handler)

	validUsernames := []string{
		"admin",
		"123456",
		"Admin123",
		"staff007",
	}

	for _, username := range validUsernames {
		t.Run(username, func(t *testing.T) {
			payload := map[string]string{
				"username": username,
				"password": "Password123!",
				"hospital": "HOSP001",
			}
			body, _ := json.Marshal(payload)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/staff/create", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				t.Errorf("expected status 201 for valid username %q, got %d", username, w.Code)
			}
		})
	}
}

func TestStaffHandler_CreateStaff_PasswordPolicy(t *testing.T) {
	mockSvc := &mockStaffService{}
	handler := NewStaffHandlerWithService(mockSvc)
	r := newTestStaffRouter(handler)

	testCases := []struct {
		name          string
		password      string
		expectedCode  int
		expectedValid bool
	}{
		// 1. Length requirements: 8 - 32 characters
		{
			name:          "too short (less than 8 chars)",
			password:      "Pass1!",
			expectedCode:  http.StatusBadRequest,
			expectedValid: false,
		},
		{
			name:          "minimum valid length (8 chars)",
			password:      "Pass12!a",
			expectedCode:  http.StatusCreated,
			expectedValid: true,
		},
		{
			name:          "maximum valid length (32 chars)",
			password:      "Pass12!aPass12!aPass12!aPass12!a",
			expectedCode:  http.StatusCreated,
			expectedValid: true,
		},
		{
			name:          "too long (more than 32 chars)",
			password:      "Pass12!aPass12!aPass12!aPass12!aExtra",
			expectedCode:  http.StatusBadRequest,
			expectedValid: false,
		},

		// 2. Missing lowercase (a-z)
		{
			name:          "missing lowercase",
			password:      "PASSWORD123!",
			expectedCode:  http.StatusBadRequest,
			expectedValid: false,
		},

		// 3. Missing uppercase (A-Z)
		{
			name:          "missing uppercase",
			password:      "password123!",
			expectedCode:  http.StatusBadRequest,
			expectedValid: false,
		},

		// 4. Missing digit (0-9)
		{
			name:          "missing digit",
			password:      "Password!@#$",
			expectedCode:  http.StatusBadRequest,
			expectedValid: false,
		},

		// 5. Missing special character
		{
			name:          "missing special character",
			password:      "Password1234",
			expectedCode:  http.StatusBadRequest,
			expectedValid: false,
		},

		// 6. Various special characters ([!@#$%^&*...])
		{
			name:          "valid with @ symbol",
			password:      "Password@123",
			expectedCode:  http.StatusCreated,
			expectedValid: true,
		},
		{
			name:          "valid with # symbol",
			password:      "Password#123",
			expectedCode:  http.StatusCreated,
			expectedValid: true,
		},
		{
			name:          "valid with $ symbol",
			password:      "Password$123",
			expectedCode:  http.StatusCreated,
			expectedValid: true,
		},
		{
			name:          "valid with % symbol",
			password:      "Password%123",
			expectedCode:  http.StatusCreated,
			expectedValid: true,
		},
		{
			name:          "valid with ^ symbol",
			password:      "Password^123",
			expectedCode:  http.StatusCreated,
			expectedValid: true,
		},
		{
			name:          "valid with & symbol",
			password:      "Password&123",
			expectedCode:  http.StatusCreated,
			expectedValid: true,
		},
		{
			name:          "valid with * symbol",
			password:      "Password*123",
			expectedCode:  http.StatusCreated,
			expectedValid: true,
		},
		{
			name:          "valid with _ symbol",
			password:      "Password_123",
			expectedCode:  http.StatusCreated,
			expectedValid: true,
		},
		{
			name:          "valid with - symbol",
			password:      "Password-123",
			expectedCode:  http.StatusCreated,
			expectedValid: true,
		},

		// 7. Non-English and invalid character rejection
		{
			name:          "contains Thai character",
			password:      "Password12ห3!",
			expectedCode:  http.StatusBadRequest,
			expectedValid: false,
		},
		{
			name:          "contains space",
			password:      "Password 123!",
			expectedCode:  http.StatusBadRequest,
			expectedValid: false,
		},
		{
			name:          "contains emoji",
			password:      "Password123!🎉",
			expectedCode:  http.StatusBadRequest,
			expectedValid: false,
		},
		{
			name:          "contains Cyrillic character",
			password:      "Password123!д",
			expectedCode:  http.StatusBadRequest,
			expectedValid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			payload := map[string]string{
				"username": "tester01",
				"password": tc.password,
				"hospital": "HOSP001",
			}
			body, _ := json.Marshal(payload)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/staff/create", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != tc.expectedCode {
				t.Errorf("expected status %d for password %q (%s), got %d. Body: %s",
					tc.expectedCode, tc.password, tc.name, w.Code, w.Body.String())
			}
		})
	}
}

func TestStaffHandler_RegisterRoutes(t *testing.T) {
	mockSvc := &mockStaffService{}
	handler := NewStaffHandlerWithService(mockSvc)
	r := gin.New()
	r.Use(middleware.ErrorHandler())

	staffGroup := r.Group("/staff")
	handler.RegisterRoutes(staffGroup)

	payload := []byte(`{"username":"admin","password":"Password123!","hospital":"HOSP001"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/staff/create", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}
}

func TestStaffHandler_IntegratedRegisterRoutes(t *testing.T) {
	mockPing := &mockPinger{err: nil}
	healthHandler := NewHealthHandler(mockPing)
	mockSvc := &mockStaffService{}
	staffHandler := NewStaffHandlerWithService(mockSvc)

	r := gin.New()
	RegisterRoutes(r, healthHandler, staffHandler)

	payload := []byte(`{"username":"admin","password":"Password123!","hospital":"HOSP001"}`)

	// Test root /staff/create
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/staff/create", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected /staff/create status 201, got %d", w.Code)
	}

	// Test root /staff/login
	loginPayload := []byte(`{"username":"admin","password":"Password123!","hospital":"HOSP001"}`)
	wLogin := httptest.NewRecorder()
	reqLogin, _ := http.NewRequest(http.MethodPost, "/staff/login", bytes.NewReader(loginPayload))
	reqLogin.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wLogin, reqLogin)

	if wLogin.Code != http.StatusOK {
		t.Errorf("expected /staff/login status 200, got %d", wLogin.Code)
	}
}

func TestStaffHandler_LoginStaff_Success(t *testing.T) {
	mockSvc := &mockStaffService{}
	handler := NewStaffHandlerWithService(mockSvc)
	r := newTestStaffRouter(handler)

	reqPayload := LoginStaffRequest{
		Username: "stafftestuser",
		Password: "SuperSecret123!",
		Hospital: "HOSP001",
	}
	body, err := json.Marshal(reqPayload)
	if err != nil {
		t.Fatalf("failed to marshal request payload: %v", err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/staff/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp response.Response[service.StaffResponseData]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("expected status 'success', got %q", resp.Status)
	}
	if resp.Message != "login successful" {
		t.Errorf("expected message 'login successful', got %q", resp.Message)
	}
	if resp.Data.Staff.Username != reqPayload.Username {
		t.Errorf("expected username %q, got %q", reqPayload.Username, resp.Data.Staff.Username)
	}
	if resp.Data.Staff.HospitalHN != "HOSP001" {
		t.Errorf("expected hospital 'HOSP001', got %q", resp.Data.Staff.HospitalHN)
	}

	// Verify Cookie
	cookieHeader := w.Header().Get("Set-Cookie")
	if !strings.Contains(cookieHeader, "session_token=mock-jwt-session-token") {
		t.Errorf("expected Set-Cookie to contain session token, got %q", cookieHeader)
	}
	if !strings.Contains(cookieHeader, "HttpOnly") {
		t.Errorf("expected Set-Cookie to have HttpOnly flag, got %q", cookieHeader)
	}
}

func TestStaffHandler_LoginStaff_MissingUsername(t *testing.T) {
	mockSvc := &mockStaffService{}
	handler := NewStaffHandlerWithService(mockSvc)
	r := newTestStaffRouter(handler)

	body := []byte(`{"password":"Password123!","hospital":"HOSP001"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/staff/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestStaffHandler_LoginStaff_MissingPassword(t *testing.T) {
	mockSvc := &mockStaffService{}
	handler := NewStaffHandlerWithService(mockSvc)
	r := newTestStaffRouter(handler)

	body := []byte(`{"username":"somchai","hospital":"HOSP001"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/staff/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestStaffHandler_LoginStaff_MissingHospital(t *testing.T) {
	mockSvc := &mockStaffService{}
	handler := NewStaffHandlerWithService(mockSvc)
	r := newTestStaffRouter(handler)

	body := []byte(`{"username":"somchai","password":"Password123!"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/staff/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 when hospital is missing, got %d", w.Code)
	}
}

func TestStaffHandler_LoginStaff_InvalidUsernameFormat(t *testing.T) {
	mockSvc := &mockStaffService{}
	handler := NewStaffHandlerWithService(mockSvc)
	r := newTestStaffRouter(handler)

	testCases := []struct {
		name     string
		username string
	}{
		{"contains_space", "staff user"},
		{"contains_special_char", "staff@user!"},
		{"contains_thai", "สมชาย"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			payload := map[string]string{
				"username": tc.username,
				"password": "Password123!",
				"hospital": "HOSP001",
			}
			body, _ := json.Marshal(payload)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/staff/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400 for username format violation, got %d", w.Code)
			}
		})
	}
}

func TestStaffHandler_LoginStaff_InvalidPasswordFormat(t *testing.T) {
	mockSvc := &mockStaffService{}
	handler := NewStaffHandlerWithService(mockSvc)
	r := newTestStaffRouter(handler)

	testCases := []struct {
		name     string
		password string
	}{
		{"too_short", "Pass1!"},
		{"missing_lowercase", "PASSWORD123!"},
		{"missing_uppercase", "password123!"},
		{"missing_digit", "Password!!!"},
		{"missing_special", "Password123"},
		{"contains_thai", "Password123!ไทย"},
		{"contains_space", "Password 123!"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			payload := map[string]string{
				"username": "somchai",
				"password": tc.password,
				"hospital": "HOSP001",
			}
			body, _ := json.Marshal(payload)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/staff/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400 for password format violation, got %d", w.Code)
			}
		})
	}
}

func TestStaffHandler_LoginStaff_InvalidCredentials(t *testing.T) {
	mockSvc := &mockStaffService{loginErr: service.ErrInvalidCredentials}
	handler := NewStaffHandlerWithService(mockSvc)
	r := newTestStaffRouter(handler)

	body := []byte(`{"username":"somchai","password":"WrongPassword123!","hospital":"HOSP001"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/staff/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d. Body: %s", w.Code, w.Body.String())
	}

	var prob middleware.ProblemDetails
	if err := json.Unmarshal(w.Body.Bytes(), &prob); err != nil {
		t.Fatalf("failed to parse problem details: %v", err)
	}
	if prob.Status != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", prob.Status)
	}
	if prob.Detail != "invalid username or password" {
		t.Errorf("expected detail 'invalid username or password', got %q", prob.Detail)
	}
	if prob.Type != "/errors/unauthorized" {
		t.Errorf("expected type '/errors/unauthorized', got %q", prob.Type)
	}
}

func TestStaffHandler_LoginStaff_HospitalNotFound(t *testing.T) {
	mockSvc := &mockStaffService{loginErr: service.ErrHospitalNotFound}
	handler := NewStaffHandlerWithService(mockSvc)
	r := newTestStaffRouter(handler)

	body := []byte(`{"username":"somchai","password":"Password123!","hospital":"NON_EXISTENT"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/staff/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 when hospital does not exist, got %d", w.Code)
	}

	var prob middleware.ProblemDetails
	if err := json.Unmarshal(w.Body.Bytes(), &prob); err != nil {
		t.Fatalf("failed to parse problem details: %v", err)
	}
	if prob.Status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", prob.Status)
	}
	if prob.Detail != "hospital not found" {
		t.Errorf("expected detail 'hospital not found', got %q", prob.Detail)
	}
	if prob.Type != "/errors/hospital-not-found" {
		t.Errorf("expected type '/errors/hospital-not-found', got %q", prob.Type)
	}
}

func TestStaffHandler_LoginStaff_InternalError(t *testing.T) {
	mockSvc := &mockStaffService{loginErr: errors.New("db error")}
	handler := NewStaffHandlerWithService(mockSvc)
	r := newTestStaffRouter(handler)

	body := []byte(`{"username":"somchai","password":"Password123!","hospital":"HOSP001"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/staff/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
}
