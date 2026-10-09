package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"hospital-middleware/internal/handlers"
	"hospital-middleware/internal/middleware"
	"hospital-middleware/internal/response"
	"hospital-middleware/internal/service"

	"github.com/gin-gonic/gin"
)

type mockPatientService struct {
	searchPatientsFunc       func(ctx context.Context, hospitalID string, input service.PatientSearchInput) (*service.PatientSearchResult, error)
	getPatientByIDFunc       func(ctx context.Context, hospitalID string, id string) (*service.PatientResponse, error)
	getPatientByIdentityFunc func(ctx context.Context, hospitalID string, identity string) (*service.PatientResponse, error)
}

func (m *mockPatientService) SearchPatients(ctx context.Context, hospitalID string, input service.PatientSearchInput) (*service.PatientSearchResult, error) {
	if m.searchPatientsFunc != nil {
		return m.searchPatientsFunc(ctx, hospitalID, input)
	}
	return nil, nil
}

func (m *mockPatientService) GetPatientByID(ctx context.Context, hospitalID string, id string) (*service.PatientResponse, error) {
	if m.getPatientByIDFunc != nil {
		return m.getPatientByIDFunc(ctx, hospitalID, id)
	}
	return nil, nil
}

func (m *mockPatientService) GetPatientByIdentity(ctx context.Context, hospitalID string, identity string) (*service.PatientResponse, error) {
	if m.getPatientByIdentityFunc != nil {
		return m.getPatientByIdentityFunc(ctx, hospitalID, identity)
	}
	if m.getPatientByIDFunc != nil {
		return m.getPatientByIDFunc(ctx, hospitalID, identity)
	}
	return nil, nil
}

func setupPatientTestRouter(svc service.PatientService, hospitalID string) *gin.Engine {
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(func(c *gin.Context) {
		if hospitalID != "" {
			c.Set(middleware.ContextKeyHospitalID, hospitalID)
		}
		c.Next()
	})

	handler := handlers.NewPatientHandler(svc)
	r.POST("/patient/search", handler.SearchPatients)
	r.GET("/patient/search/:id", handler.GetPatientByID)
	return r
}

func TestPatientHandler_MissingHospitalContext(t *testing.T) {
	r := setupPatientTestRouter(&mockPatientService{}, "") // No hospital ID

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/patient/search?first_name=Alice", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 when hospital context is missing, got %d", w.Code)
	}

	var problem middleware.ProblemDetails
	if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
		t.Fatalf("failed to parse problem details: %v", err)
	}
	if problem.Status != http.StatusUnauthorized {
		t.Errorf("expected problem status 401, got %d", problem.Status)
	}
}

func TestPatientHandler_SearchViaQueryParams_Success(t *testing.T) {
	var capturedInput service.PatientSearchInput
	var capturedHospID string

	mockSvc := &mockPatientService{
		searchPatientsFunc: func(ctx context.Context, hospitalID string, input service.PatientSearchInput) (*service.PatientSearchResult, error) {
			capturedHospID = hospitalID
			capturedInput = input
			return &service.PatientSearchResult{
				Patients: []service.PatientResponse{
					{
						ID:          "p-1",
						PatientHN:   "HN001",
						FirstNameEN: "Somchai",
						LastNameEN:  "Jaidee",
						DateOfBirth: "1990-05-15",
						Gender:      "M",
					},
				},
				Total: 1,
			}, nil
		},
	}

	r := setupPatientTestRouter(mockSvc, "hosp-123")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/patient/search?first_name=Somchai&last_name=Jaidee", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	if capturedHospID != "hosp-123" {
		t.Errorf("expected hospital ID 'hosp-123', got %s", capturedHospID)
	}
	if capturedInput.FirstName != "Somchai" || capturedInput.LastName != "Jaidee" {
		t.Errorf("unexpected input captured: %+v", capturedInput)
	}

	var resp response.Response[service.PatientSearchResult]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "success" {
		t.Errorf("expected status 'success', got %s", resp.Status)
	}
	if resp.Data.Total != 1 || len(resp.Data.Patients) != 1 {
		t.Errorf("expected 1 patient, got %d", len(resp.Data.Patients))
	}
}

func TestPatientHandler_SearchViaJSONBody_Success(t *testing.T) {
	var capturedInput service.PatientSearchInput

	mockSvc := &mockPatientService{
		searchPatientsFunc: func(ctx context.Context, hospitalID string, input service.PatientSearchInput) (*service.PatientSearchResult, error) {
			capturedInput = input
			return &service.PatientSearchResult{
				Patients: []service.PatientResponse{},
				Total:    0,
			}, nil
		},
	}

	r := setupPatientTestRouter(mockSvc, "hosp-123")

	payload := `{"national_id":"1100501234567"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/patient/search", bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	if capturedInput.NationalID != "1100501234567" {
		t.Errorf("expected NationalID '1100501234567', got %s", capturedInput.NationalID)
	}
}

func TestPatientHandler_EmptyCriteria_BadRequest(t *testing.T) {
	mockSvc := &mockPatientService{
		searchPatientsFunc: func(ctx context.Context, hospitalID string, input service.PatientSearchInput) (*service.PatientSearchResult, error) {
			return nil, service.ErrEmptySearchCriteria
		},
	}

	r := setupPatientTestRouter(mockSvc, "hosp-123")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/patient/search", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for empty criteria, got %d", w.Code)
	}

	var problem middleware.ProblemDetails
	if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
		t.Fatalf("failed to parse problem details: %v", err)
	}
	if problem.Detail != service.ErrEmptySearchCriteria.Error() {
		t.Errorf("expected detail %q, got %q", service.ErrEmptySearchCriteria.Error(), problem.Detail)
	}
}

func TestPatientHandler_InvalidDateOfBirth_BadRequest(t *testing.T) {
	mockSvc := &mockPatientService{
		searchPatientsFunc: func(ctx context.Context, hospitalID string, input service.PatientSearchInput) (*service.PatientSearchResult, error) {
			return nil, service.ErrInvalidDateOfBirth
		},
	}

	r := setupPatientTestRouter(mockSvc, "hosp-123")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/patient/search?date_of_birth=invalid-date", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid date, got %d", w.Code)
	}
}


func TestPatientHandler_MalformedJSON_BadRequest(t *testing.T) {
	mockSvc := &mockPatientService{}
	r := setupPatientTestRouter(mockSvc, "hosp-123")

	malformedJSON := `{"first_name": "Alice",,}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/patient/search", bytes.NewReader([]byte(malformedJSON)))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for malformed json, got %d", w.Code)
	}
}

func TestPatientHandler_ServiceInternalError(t *testing.T) {
	mockSvc := &mockPatientService{
		searchPatientsFunc: func(ctx context.Context, hospitalID string, input service.PatientSearchInput) (*service.PatientSearchResult, error) {
			return nil, errors.New("db query timeout")
		},
	}

	r := setupPatientTestRouter(mockSvc, "hosp-123")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/patient/search?first_name=Alice", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500 for service internal error, got %d", w.Code)
	}
}

func TestPatientHandler_GetPatientByID_MissingHospitalContext(t *testing.T) {
	r := setupPatientTestRouter(&mockPatientService{}, "") // No hospital ID

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/patient/search/e7b23c21-1234-4567-89ab-cdef01234567", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 when hospital context is missing, got %d", w.Code)
	}

	var problem middleware.ProblemDetails
	if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
		t.Fatalf("failed to parse problem details: %v", err)
	}
	if problem.Status != http.StatusUnauthorized {
		t.Errorf("expected problem status 401, got %d", problem.Status)
	}
}

func TestPatientHandler_GetPatientByID_InvalidIDFormat(t *testing.T) {
	r := setupPatientTestRouter(&mockPatientService{}, "hosp-123")

	invalidIDs := []string{
		"123",
		"abc",
		"invalid-id!",
		"e7b23c21-1234-4567-89ab-cdef01234567", // UUID rejected: must be national_id or passport_id
		"1234567890123456789012345678901",
	}

	for _, id := range invalidIDs {
		t.Run("invalid_id_"+id, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/patient/search/"+id, nil)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400 for invalid ID %q, got %d. Body: %s", id, w.Code, w.Body.String())
			}

			var problem middleware.ProblemDetails
			if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
				t.Fatalf("failed to parse problem details: %v", err)
			}
			if problem.Status != http.StatusBadRequest {
				t.Errorf("expected problem status 400, got %d", problem.Status)
			}
			if problem.Detail != service.ErrInvalidPatientID.Error() {
				t.Errorf("expected detail %q, got %q", service.ErrInvalidPatientID.Error(), problem.Detail)
			}
		})
	}
}

func TestPatientHandler_GetPatientByID_NotFound(t *testing.T) {
	mockSvc := &mockPatientService{
		getPatientByIDFunc: func(ctx context.Context, hospitalID string, id string) (*service.PatientResponse, error) {
			return nil, service.ErrPatientNotFound
		},
	}

	r := setupPatientTestRouter(mockSvc, "hosp-123")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/patient/search/9999999999999", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for non-existent patient, got %d. Body: %s", w.Code, w.Body.String())
	}

	var problem middleware.ProblemDetails
	if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
		t.Fatalf("failed to parse problem details: %v", err)
	}
	if problem.Status != http.StatusNotFound {
		t.Errorf("expected problem status 404, got %d", problem.Status)
	}
	if problem.Detail != "patient not found" {
		t.Errorf("expected detail 'patient not found', got %q", problem.Detail)
	}
}

func TestPatientHandler_GetPatientByID_Success(t *testing.T) {
	mockPatient := &service.PatientResponse{
		ID:           "e7b23c21-1234-4567-89ab-cdef01234567",
		PatientHN:    "P00020",
		FirstNameTH:  "ชลธิชา",
		MiddleNameTH: "",
		LastNameTH:   "สว่างจิตต์",
		FirstNameEN:  "Chonthicha",
		MiddleNameEN: "",
		LastNameEN:   "Sawangjit",
		DateOfBirth:  "2004-10-15",
		NationalID:   "1100505566779",
		PassportID:   "US556677889",
		PhoneNumber:  "0832223333",
		Email:        "chonthicha.s@example.com",
		Gender:       "F",
	}

	mockSvc := &mockPatientService{
		getPatientByIDFunc: func(ctx context.Context, hospitalID string, id string) (*service.PatientResponse, error) {
			if hospitalID == "hosp-123" && (id == "1100505566779" || id == "US556677889") {
				return mockPatient, nil
			}
			return nil, service.ErrPatientNotFound
		},
	}

	r := setupPatientTestRouter(mockSvc, "hosp-123")

	// 1. Test retrieval by National ID
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodGet, "/patient/search/1100505566779", nil)
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("expected status 200 searching by national ID, got %d. Body: %s", w1.Code, w1.Body.String())
	}

	var resp1 response.Response[service.PatientResponse]
	if err := json.Unmarshal(w1.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}
	if resp1.Data.NationalID != "1100505566779" {
		t.Errorf("expected NationalID 1100505566779, got %s", resp1.Data.NationalID)
	}

	// 2. Test retrieval by Passport ID
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/patient/search/US556677889", nil)
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected status 200 searching by passport ID, got %d. Body: %s", w2.Code, w2.Body.String())
	}

	var resp2 response.Response[service.PatientResponse]
	if err := json.Unmarshal(w2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}
	if resp2.Data.PassportID != "US556677889" {
		t.Errorf("expected PassportID US556677889, got %s", resp2.Data.PassportID)
	}
}

func TestPatientHandler_GetPatientByID_ServiceInternalError(t *testing.T) {
	mockSvc := &mockPatientService{
		getPatientByIDFunc: func(ctx context.Context, hospitalID string, id string) (*service.PatientResponse, error) {
			return nil, errors.New("db query timeout")
		},
	}

	r := setupPatientTestRouter(mockSvc, "hosp-123")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/patient/search/1100505566779", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500 for service internal error, got %d", w.Code)
	}
}
