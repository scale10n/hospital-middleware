package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hospital-middleware/internal/config"
	"hospital-middleware/internal/database"
	"hospital-middleware/internal/middleware"
	"hospital-middleware/internal/models"
	"hospital-middleware/internal/repository"
	"hospital-middleware/internal/response"
	"hospital-middleware/internal/service"

	"github.com/gin-gonic/gin"
)

type searchByIDFixtures struct {
	router      *gin.Engine
	cookieHosp1 *http.Cookie
	cookieHosp2 *http.Cookie
	hosp1       *models.Hospital
	hosp2       *models.Hospital
	patient1    *models.Patient
	patient2    *models.Patient
	patient3    *models.Patient
	teardown    func()
}

func setupSearchByIDTestFixtures(t *testing.T) *searchByIDFixtures {
	t.Helper()

	cfg := config.Load()
	db, err := database.New(cfg.DB)
	if err != nil {
		t.Skipf("skipping test: database connection not available: %v", err)
	}

	ctx := context.Background()
	router := setupRouter(db, cfg)

	hospRepo := repository.NewHospitalRepository(db)
	patientRepo := repository.NewPatientRepository(db)

	hosp1, err := hospRepo.FindByHN(ctx, "HOSP001")
	if err != nil || hosp1 == nil {
		db.Close()
		t.Skipf("skipping test: hospital HOSP001 not found: %v", err)
	}
	hosp2, err := hospRepo.FindByHN(ctx, "HOSP002")
	if err != nil || hosp2 == nil {
		db.Close()
		t.Skipf("skipping test: hospital HOSP002 not found: %v", err)
	}

	staff1User := "byidstaff1"
	staff2User := "byidstaff2"
	_, _ = db.ExecContext(ctx, "DELETE FROM staff WHERE username IN ($1, $2)", staff1User, staff2User)

	createStaffCookie := func(user, pass, hospHN string) *http.Cookie {
		regPayload := fmt.Sprintf(`{"username":%q,"password":%q,"hospital":%q}`, user, pass, hospHN)
		wReg := httptest.NewRecorder()
		rReg, _ := http.NewRequest(http.MethodPost, "/staff/create", bytes.NewReader([]byte(regPayload)))
		rReg.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(wReg, rReg)
		if wReg.Code != http.StatusCreated {
			t.Fatalf("failed to create staff %s: %s", user, wReg.Body.String())
		}
		for _, cookie := range wReg.Result().Cookies() {
			if cookie.Name == middleware.SessionCookieName {
				return cookie
			}
		}
		t.Fatalf("session cookie not found for %s", user)
		return nil
	}

	cookie1 := createStaffCookie(staff1User, "Password123!", "HOSP001")
	cookie2 := createStaffCookie(staff2User, "Password123!", "HOSP002")

	dob1, _ := time.Parse("2006-01-02", "1988-11-23")
	dob2, _ := time.Parse("2006-01-02", "1994-04-12")

	// Patient 1 in HOSP001 - Thai citizen with both NationalID and PassportID
	p1 := &models.Patient{
		HospitalID:   hosp1.ID,
		PatientHN:    "HN-BYID-001",
		NationalID:   "8800112233001",
		PassportID:   "BYIDPASS01",
		FirstNameTH:  "อนันต์",
		MiddleNameTH: "ชัยวัฒน์",
		LastNameTH:   "สุขสมบูรณ์",
		FirstNameEN:  "Anan",
		MiddleNameEN: "Chaiwat",
		LastNameEN:   "Suksomboon",
		DateOfBirth:  dob1,
		PhoneNumber:  "0819870001",
		Email:        "anan.s@hosp1.example.com",
		Gender:       models.GenderMale,
	}

	// Patient 2 in HOSP001 - Thai citizen with only NationalID (no PassportID)
	p2 := &models.Patient{
		HospitalID:   hosp1.ID,
		PatientHN:    "HN-BYID-002",
		NationalID:   "8800112233002",
		PassportID:   "",
		FirstNameTH:  "กานดา",
		MiddleNameTH: "",
		LastNameTH:   "วงศ์วิเศษ",
		FirstNameEN:  "Kanda",
		MiddleNameEN: "",
		LastNameEN:   "Wongwiset",
		DateOfBirth:  dob2,
		PhoneNumber:  "0829870002",
		Email:        "kanda.w@hosp1.example.com",
		Gender:       models.GenderFemale,
	}

	// Patient 3 in HOSP002 - Foreign patient in HOSP002 with only PassportID
	p3 := &models.Patient{
		HospitalID:   hosp2.ID,
		PatientHN:    "HN-BYID-003",
		NationalID:   "",
		PassportID:   "BYIDPASS03",
		FirstNameTH:  "เดวิด",
		MiddleNameTH: "",
		LastNameTH:   "สมิธ",
		FirstNameEN:  "David",
		MiddleNameEN: "",
		LastNameEN:   "Smith",
		DateOfBirth:  dob1,
		PhoneNumber:  "0839870003",
		Email:        "david.smith@hosp2.example.com",
		Gender:       models.GenderMale,
	}

	cleanup := func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM patient WHERE patient_hn IN ('HN-BYID-001', 'HN-BYID-002', 'HN-BYID-003')")
	}
	cleanup()

	created1, err := patientRepo.Create(ctx, p1)
	if err != nil {
		t.Fatalf("failed to create fixture patient 1: %v", err)
	}
	created2, err := patientRepo.Create(ctx, p2)
	if err != nil {
		t.Fatalf("failed to create fixture patient 2: %v", err)
	}
	created3, err := patientRepo.Create(ctx, p3)
	if err != nil {
		t.Fatalf("failed to create fixture patient 3: %v", err)
	}

	teardown := func() {
		_ = patientRepo.DeleteByID(ctx, created1.ID)
		_ = patientRepo.DeleteByID(ctx, created2.ID)
		_ = patientRepo.DeleteByID(ctx, created3.ID)
		cleanup()
		_, _ = db.ExecContext(ctx, "DELETE FROM staff WHERE username IN ($1, $2)", staff1User, staff2User)
		db.Close()
	}

	return &searchByIDFixtures{
		router:      router,
		cookieHosp1: cookie1,
		cookieHosp2: cookie2,
		hosp1:       hosp1,
		hosp2:       hosp2,
		patient1:    created1,
		patient2:    created2,
		patient3:    created3,
		teardown:    teardown,
	}
}

// TestPatientSearchByID_Success verifies retrieval via national_id or passport_id
func TestPatientSearchByID_Success(t *testing.T) {
	f := setupSearchByIDTestFixtures(t)
	defer f.teardown()

	// 1. Search by 13-digit National ID
	t.Run("by_national_id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/patient/search/"+f.patient1.NationalID, nil)
		req.AddCookie(f.cookieHosp1)

		f.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 searching by national_id, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp response.Response[service.PatientResponse]
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Status != "success" {
			t.Errorf("expected status 'success', got %s", resp.Status)
		}

		p := resp.Data
		if p.ID != f.patient1.ID {
			t.Errorf("expected ID %s, got %s", f.patient1.ID, p.ID)
		}
		if p.NationalID != f.patient1.NationalID {
			t.Errorf("expected NationalID %s, got %s", f.patient1.NationalID, p.NationalID)
		}
		if p.PatientHN != "HN-BYID-001" {
			t.Errorf("expected HN HN-BYID-001, got %s", p.PatientHN)
		}
		if p.FirstNameTH != "อนันต์" || p.LastNameTH != "สุขสมบูรณ์" {
			t.Errorf("unexpected Thai names: %+v", p)
		}
		if p.FirstNameEN != "Anan" || p.LastNameEN != "Suksomboon" {
			t.Errorf("unexpected English names: %+v", p)
		}
		if p.DateOfBirth != "1988-11-23" {
			t.Errorf("expected DOB 1988-11-23, got %s", p.DateOfBirth)
		}
		if p.Gender != "M" {
			t.Errorf("expected gender M, got %s", p.Gender)
		}
	})

	// 2. Search by Passport ID
	t.Run("by_passport_id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/patient/search/"+f.patient1.PassportID, nil)
		req.AddCookie(f.cookieHosp1)

		f.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 searching by passport_id, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp response.Response[service.PatientResponse]
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Data.ID != f.patient1.ID || resp.Data.PassportID != f.patient1.PassportID {
			t.Errorf("expected patient ID %s and PassportID %s, got %+v", f.patient1.ID, f.patient1.PassportID, resp.Data)
		}
	})

	// 3. Search by lowercase Passport ID (case-insensitive)
	t.Run("by_passport_id_case_insensitive", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/patient/search/"+strings.ToLower(f.patient1.PassportID), nil)
		req.AddCookie(f.cookieHosp1)

		f.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 for case-insensitive passport search, got %d. Body: %s", w.Code, w.Body.String())
		}
	})
}

// TestPatientSearchByID_EmptyOptionalFields verifies correct handling of empty optional fields
func TestPatientSearchByID_EmptyOptionalFields(t *testing.T) {
	f := setupSearchByIDTestFixtures(t)
	defer f.teardown()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/patient/search/"+f.patient2.NationalID, nil)
	req.AddCookie(f.cookieHosp1)

	f.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp response.Response[service.PatientResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	p := resp.Data
	if p.ID != f.patient2.ID {
		t.Errorf("expected ID %s, got %s", f.patient2.ID, p.ID)
	}
	if p.MiddleNameTH != "" || p.MiddleNameEN != "" {
		t.Errorf("expected empty middle names, got TH=%q EN=%q", p.MiddleNameTH, p.MiddleNameEN)
	}
	if p.PassportID != "" {
		t.Errorf("expected empty passport ID, got %q", p.PassportID)
	}
	if p.Gender != "F" {
		t.Errorf("expected gender F, got %s", p.Gender)
	}
}

// TestPatientSearchByID_NotFound verifies 404 for valid identity format not in database
func TestPatientSearchByID_NotFound(t *testing.T) {
	f := setupSearchByIDTestFixtures(t)
	defer f.teardown()

	t.Run("non_existent_national_id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/patient/search/9999999999999", nil)
		req.AddCookie(f.cookieHosp1)

		f.router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404 for non-existent national_id, got %d. Body: %s", w.Code, w.Body.String())
		}

		var problem middleware.ProblemDetails
		if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
			t.Fatalf("failed to decode ProblemDetails: %v", err)
		}
		if problem.Status != http.StatusNotFound || problem.Detail != "patient not found" {
			t.Errorf("expected 404 patient not found, got %+v", problem)
		}
	})

	t.Run("non_existent_passport_id", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/patient/search/US998877665", nil)
		req.AddCookie(f.cookieHosp1)

		f.router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404 for non-existent passport_id, got %d. Body: %s", w.Code, w.Body.String())
		}
	})
}

// TestPatientSearchByID_MultiTenantIsolation verifies 404 when querying patient belonging to another hospital
func TestPatientSearchByID_MultiTenantIsolation(t *testing.T) {
	f := setupSearchByIDTestFixtures(t)
	defer f.teardown()

	// Staff from HOSP002 attempts to query patient 1's national_id (belongs to HOSP001)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/patient/search/"+f.patient1.NationalID, nil)
	req.AddCookie(f.cookieHosp2)

	f.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 due to hospital isolation, got %d. Body: %s", w.Code, w.Body.String())
	}

	var problem middleware.ProblemDetails
	if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
		t.Fatalf("failed to decode ProblemDetails: %v", err)
	}
	if problem.Status != http.StatusNotFound {
		t.Errorf("expected problem status 404, got %d", problem.Status)
	}
}

// TestPatientSearchByID_InvalidIDFormats verifies 400 Bad Request for malformed identity formats
func TestPatientSearchByID_InvalidIDFormats(t *testing.T) {
	f := setupSearchByIDTestFixtures(t)
	defer f.teardown()

	invalidIDs := []struct {
		name string
		id   string
	}{
		{name: "short string", id: "123"},
		{name: "short alpha", id: "abc"},
		{name: "patient UUID with hyphens rejected", id: f.patient1.ID},
		{name: "special characters", id: "110050123456!"},
		{name: "sql injection attempt", id: "' OR 1=1; --"},
		{name: "spaced characters", id: "1100 5012 34567"},
		{name: "too long string", id: "1234567890123456789012345678901"},
	}

	for _, tc := range invalidIDs {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/patient/search/"+tc.id, nil)
			req.AddCookie(f.cookieHosp1)

			f.router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400 for invalid ID %q, got %d. Body: %s", tc.id, w.Code, w.Body.String())
			}

			var problem middleware.ProblemDetails
			if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
				t.Fatalf("failed to decode ProblemDetails: %v", err)
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

// TestPatientSearchByID_AuthenticationFailures verifies 401 Unauthorized
func TestPatientSearchByID_AuthenticationFailures(t *testing.T) {
	f := setupSearchByIDTestFixtures(t)
	defer f.teardown()

	// 1. Missing cookie
	t.Run("missing cookie", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/patient/search/"+f.patient1.NationalID, nil)

		f.router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401 for missing cookie, got %d", w.Code)
		}
	})

	// 2. Invalid cookie value
	t.Run("invalid cookie token", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/patient/search/"+f.patient1.NationalID, nil)
		req.AddCookie(&http.Cookie{
			Name:  middleware.SessionCookieName,
			Value: "garbage-jwt-token-value",
		})

		f.router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401 for invalid cookie token, got %d", w.Code)
		}
	})
}

// TestPatientSearchByID_Performance verifies response time < 200ms
func TestPatientSearchByID_Performance(t *testing.T) {
	f := setupSearchByIDTestFixtures(t)
	defer f.teardown()

	const iterations = 10
	var totalDuration time.Duration

	for i := 0; i < iterations; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/patient/search/"+f.patient1.NationalID, nil)
		req.AddCookie(f.cookieHosp1)

		start := time.Now()
		f.router.ServeHTTP(w, req)
		elapsed := time.Since(start)

		totalDuration += elapsed

		if w.Code != http.StatusOK {
			t.Errorf("iteration %d: expected status 200, got %d", i+1, w.Code)
		}
		if elapsed >= 200*time.Millisecond {
			t.Errorf("iteration %d: expected response time < 200ms, got %v", i+1, elapsed)
		}
	}

	avgDuration := totalDuration / iterations
	t.Logf("GET /patient/search/{id} average response time over %d iterations: %v (target: < 200ms)", iterations, avgDuration)
}

// TestPatientSearchByID_TableDriven_IdentityScenarios implements table-driven tests for:
// - Valid National ID (matching patient) -> 200 OK
// - Valid Passport ID (matching patient) -> 200 OK
// - Valid identity format with no matching patient -> 404 Not Found
// - Invalid identity formats -> 400 Bad Request
func TestPatientSearchByID_TableDriven_IdentityScenarios(t *testing.T) {
	f := setupSearchByIDTestFixtures(t)
	defer f.teardown()

	tests := []struct {
		name           string
		patientID      string
		expectedStatus int
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name:           "Valid 13-digit National ID with existing patient",
			patientID:      f.patient1.NationalID,
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp response.Response[service.PatientResponse]
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to parse success body: %v", err)
				}
				if resp.Status != "success" || resp.Data.NationalID != f.patient1.NationalID {
					t.Errorf("expected success with NationalID %s, got %+v", f.patient1.NationalID, resp)
				}
			},
		},
		{
			name:           "Valid Passport ID with existing patient",
			patientID:      f.patient1.PassportID,
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp response.Response[service.PatientResponse]
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to parse success body: %v", err)
				}
				if resp.Status != "success" || resp.Data.PassportID != f.patient1.PassportID {
					t.Errorf("expected success with PassportID %s, got %+v", f.patient1.PassportID, resp)
				}
			},
		},
		{
			name:           "Valid lowercase Passport ID with existing patient",
			patientID:      strings.ToLower(f.patient1.PassportID),
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp response.Response[service.PatientResponse]
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to parse success body: %v", err)
				}
				if resp.Status != "success" {
					t.Errorf("expected success, got %+v", resp)
				}
			},
		},
		{
			name:           "Valid 13-digit National ID with no matching patient",
			patientID:      "9999999999999",
			expectedStatus: http.StatusNotFound,
			checkBody: func(t *testing.T, body []byte) {
				var prob middleware.ProblemDetails
				if err := json.Unmarshal(body, &prob); err != nil {
					t.Fatalf("failed to parse error body: %v", err)
				}
				if prob.Status != http.StatusNotFound || prob.Detail != "patient not found" {
					t.Errorf("expected 404 patient not found, got %+v", prob)
				}
			},
		},
		{
			name:           "Valid Passport ID format with no matching patient",
			patientID:      "XX998877665",
			expectedStatus: http.StatusNotFound,
			checkBody: func(t *testing.T, body []byte) {
				var prob middleware.ProblemDetails
				if err := json.Unmarshal(body, &prob); err != nil {
					t.Fatalf("failed to parse error body: %v", err)
				}
				if prob.Status != http.StatusNotFound {
					t.Errorf("expected 404, got %+v", prob)
				}
			},
		},
		{
			name:           "Invalid ID - too short digits",
			patientID:      "123",
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body []byte) {
				var prob middleware.ProblemDetails
				if err := json.Unmarshal(body, &prob); err != nil {
					t.Fatalf("failed to parse error body: %v", err)
				}
				if prob.Status != http.StatusBadRequest || prob.Detail != service.ErrInvalidPatientID.Error() {
					t.Errorf("expected 400 invalid format, got %+v", prob)
				}
			},
		},
		{
			name:           "Invalid ID - too short letters",
			patientID:      "abc",
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body []byte) {
				var prob middleware.ProblemDetails
				_ = json.Unmarshal(body, &prob)
				if prob.Status != http.StatusBadRequest {
					t.Errorf("expected 400, got %+v", prob)
				}
			},
		},
		{
			name:           "Invalid ID - Patient UUID with hyphens is rejected",
			patientID:      f.patient1.ID,
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body []byte) {
				var prob middleware.ProblemDetails
				_ = json.Unmarshal(body, &prob)
				if prob.Status != http.StatusBadRequest {
					t.Errorf("expected 400 rejecting UUID, got %+v", prob)
				}
			},
		},
		{
			name:           "Invalid ID - symbols and punctuation",
			patientID:      "110050123456!",
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body []byte) {
				var prob middleware.ProblemDetails
				_ = json.Unmarshal(body, &prob)
				if prob.Status != http.StatusBadRequest {
					t.Errorf("expected 400, got %+v", prob)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/patient/search/"+tc.patientID, nil)
			req.AddCookie(f.cookieHosp1)

			f.router.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d. Body: %s", tc.expectedStatus, w.Code, w.Body.String())
			}

			if tc.checkBody != nil {
				tc.checkBody(t, w.Body.Bytes())
			}
		})
	}
}

// TestPatientSearchByID_Security_AuthAndSQLInjection validates authentication enforcement and SQL injection protection
func TestPatientSearchByID_Security_AuthAndSQLInjection(t *testing.T) {
	f := setupSearchByIDTestFixtures(t)
	defer f.teardown()

	// 1. Authentication enforcement tests
	t.Run("Auth_MissingSessionCookie", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/patient/search/"+f.patient1.NationalID, nil)
		f.router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("Auth_InvalidSessionToken", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/patient/search/"+f.patient1.NationalID, nil)
		req.AddCookie(&http.Cookie{
			Name:  middleware.SessionCookieName,
			Value: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid.signature",
		})
		f.router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("Auth_MultiTenantIsolation_AccessFromOtherHospitalDenied", func(t *testing.T) {
		// Staff from HOSP002 querying HOSP001 patient
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/patient/search/"+f.patient1.NationalID, nil)
		req.AddCookie(f.cookieHosp2)
		f.router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404 Not Found for cross-hospital access, got %d", w.Code)
		}
	})

	// 2. SQL injection prevention tests
	sqlInjectionPayloads := []struct {
		name    string
		payload string
	}{
		{name: "classic boolean OR injection", payload: "' OR '1'='1"},
		{name: "numeric boolean OR injection", payload: "1' OR 1=1 --"},
		{name: "valid ID prefix with OR injection", payload: f.patient1.NationalID + "' OR '1'='1"},
		{name: "stacked query drop table attempt", payload: "'; DROP TABLE patient; --"},
		{name: "union select injection attempt", payload: "' UNION SELECT gen_random_uuid(), gen_random_uuid(), 'HACK', 'HACK' --"},
		{name: "time delay sleep injection attempt", payload: "1; SELECT pg_sleep(3); --"},
		{name: "comment dash injection", payload: "admin'--"},
		{name: "escaped quote injection", payload: `\' OR 1=1 --`},
	}

	for _, tc := range sqlInjectionPayloads {
		t.Run("SQLInj_"+tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/patient/search/"+tc.payload, nil)
			req.AddCookie(f.cookieHosp1)

			f.router.ServeHTTP(w, req)

			// SQL injection attempts MUST be rejected with 400 Bad Request due to strict identity validation
			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 Bad Request for SQL injection attempt %q, got %d. Body: %s", tc.payload, w.Code, w.Body.String())
			}

			var prob middleware.ProblemDetails
			if err := json.Unmarshal(w.Body.Bytes(), &prob); err != nil {
				t.Fatalf("failed to decode ProblemDetails: %v", err)
			}
			if prob.Status != http.StatusBadRequest {
				t.Errorf("expected problem status 400, got %d", prob.Status)
			}
			if prob.Detail != service.ErrInvalidPatientID.Error() {
				t.Errorf("expected detail %q, got %q", service.ErrInvalidPatientID.Error(), prob.Detail)
			}
		})
	}
}
