package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"hospital-middleware/internal/auth"
	"hospital-middleware/internal/config"
	"hospital-middleware/internal/database"
	"hospital-middleware/internal/middleware"
	"hospital-middleware/internal/models"
	"hospital-middleware/internal/repository"
	"hospital-middleware/internal/response"
	"hospital-middleware/internal/service"

	"github.com/gin-gonic/gin"
)

// setupChecklistFixtures prepares isolated test data across HOSP001 and HOSP002.
func setupChecklistFixtures(t *testing.T) (*gin.Engine, *http.Cookie, *http.Cookie, *models.Hospital, *models.Hospital, func()) {
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

	// 1. Setup authenticated staff for HOSP001 and HOSP002
	staff1User := "chkstaff1"
	staff2User := "chkstaff2"
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

	// 2. Setup isolated test patients with globally distinct names and dates
	dobA, _ := time.Parse("2006-01-02", "1973-03-21")
	dobB, _ := time.Parse("2006-01-02", "1978-06-14")

	// Patient A in HOSP001
	patientA := &models.Patient{
		HospitalID:   hosp1.ID,
		PatientHN:    "HN-CHK-001",
		NationalID:   "9900112233001",
		PassportID:   "CHKPASS001",
		FirstNameTH:  "เช็กเกอร์กิตติ",
		MiddleNameTH: "เช็กเกอร์รุ่งเรือง",
		LastNameTH:   "เช็กเกอร์เจริญยิ่ง",
		FirstNameEN:  "CheckAlpha",
		MiddleNameEN: "CheckRungruang",
		LastNameEN:   "CheckCharoenying",
		DateOfBirth:  dobA,
		PhoneNumber:  "0812340001",
		Email:        "alpha.chk@hospital.example.com",
		Gender:       models.GenderMale,
	}

	// Patient B in HOSP001 (shares last name and "Check" prefix with A, but distinct first name)
	patientB := &models.Patient{
		HospitalID:   hosp1.ID,
		PatientHN:    "HN-CHK-002",
		NationalID:   "9900112233002",
		PassportID:   "CHKPASS002",
		FirstNameTH:  "เช็กเกอร์พานิช",
		MiddleNameTH: "",
		LastNameTH:   "เช็กเกอร์เจริญยิ่ง",
		FirstNameEN:  "CheckBeta",
		MiddleNameEN: "",
		LastNameEN:   "CheckCharoenying",
		DateOfBirth:  dobB,
		PhoneNumber:  "0812340002",
		Email:        "beta.chk@hospital.example.com",
		Gender:       models.GenderFemale,
	}

	// Patient C in HOSP002 (clone names & DOB of Patient A, but scoped strictly to HOSP002)
	patientC := &models.Patient{
		HospitalID:   hosp2.ID,
		PatientHN:    "HN-CHK-003",
		NationalID:   "9900112233003",
		PassportID:   "CHKPASS003",
		FirstNameTH:  "เช็กเกอร์กิตติ",
		MiddleNameTH: "เช็กเกอร์รุ่งเรือง",
		LastNameTH:   "เช็กเกอร์เจริญยิ่ง",
		FirstNameEN:  "CheckAlpha",
		MiddleNameEN: "CheckRungruang",
		LastNameEN:   "CheckCharoenying",
		DateOfBirth:  dobA,
		PhoneNumber:  "0823450003",
		Email:        "alpha.chula@hospital.example.com",
		Gender:       models.GenderMale,
	}

	cleanupPatients := func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM patient WHERE patient_hn IN ('HN-CHK-001', 'HN-CHK-002', 'HN-CHK-003')")
	}
	cleanupPatients()

	createdA, err := patientRepo.Create(ctx, patientA)
	if err != nil {
		t.Fatalf("failed to create fixture patient A: %v", err)
	}
	createdB, err := patientRepo.Create(ctx, patientB)
	if err != nil {
		t.Fatalf("failed to create fixture patient B: %v", err)
	}
	createdC, err := patientRepo.Create(ctx, patientC)
	if err != nil {
		t.Fatalf("failed to create fixture patient C: %v", err)
	}

	teardown := func() {
		_ = patientRepo.DeleteByID(ctx, createdA.ID)
		_ = patientRepo.DeleteByID(ctx, createdB.ID)
		_ = patientRepo.DeleteByID(ctx, createdC.ID)
		cleanupPatients()
		_, _ = db.ExecContext(ctx, "DELETE FROM staff WHERE username IN ($1, $2)", staff1User, staff2User)
		db.Close()
	}

	return router, cookie1, cookie2, hosp1, hosp2, teardown
}

// ============================================================================
// 1. Table-Driven Tests: Single criterion (national_id only, name only, etc.)
// ============================================================================
func TestPatientSearch_TableDriven_SingleCriterion(t *testing.T) {
	router, cookie1, _, _, _, teardown := setupChecklistFixtures(t)
	defer teardown()

	tests := []struct {
		name          string
		queryKey      string
		queryValue    string
		expectedTotal int
		expectedHN    string
	}{
		{
			name:          "single criterion: national_id only",
			queryKey:      "national_id",
			queryValue:    "9900112233001",
			expectedTotal: 1,
			expectedHN:    "HN-CHK-001",
		},
		{
			name:          "single criterion: passport_id only",
			queryKey:      "passport_id",
			queryValue:    "CHKPASS001",
			expectedTotal: 1,
			expectedHN:    "HN-CHK-001",
		},
		{
			name:          "single criterion: first_name English only",
			queryKey:      "first_name",
			queryValue:    "CheckAlpha",
			expectedTotal: 1,
			expectedHN:    "HN-CHK-001",
		},
		{
			name:          "single criterion: first_name Thai only",
			queryKey:      "first_name",
			queryValue:    "เช็กเกอร์กิตติ",
			expectedTotal: 1,
			expectedHN:    "HN-CHK-001",
		},
		{
			name:          "single criterion: middle_name English only",
			queryKey:      "middle_name",
			queryValue:    "CheckRungruang",
			expectedTotal: 1,
			expectedHN:    "HN-CHK-001",
		},
		{
			name:          "single criterion: middle_name Thai only",
			queryKey:      "middle_name",
			queryValue:    "เช็กเกอร์รุ่งเรือง",
			expectedTotal: 1,
			expectedHN:    "HN-CHK-001",
		},
		{
			name:          "single criterion: last_name English only (matches both A and B)",
			queryKey:      "last_name",
			queryValue:    "CheckCharoenying",
			expectedTotal: 2,
			expectedHN:    "", // Multiple results
		},
		{
			name:          "single criterion: last_name Thai only (matches both A and B)",
			queryKey:      "last_name",
			queryValue:    "เช็กเกอร์เจริญยิ่ง",
			expectedTotal: 2,
			expectedHN:    "",
		},
		{
			name:          "single criterion: date_of_birth only",
			queryKey:      "date_of_birth",
			queryValue:    "1973-03-21",
			expectedTotal: 1,
			expectedHN:    "HN-CHK-001",
		},
		{
			name:          "single criterion: phone_number only",
			queryKey:      "phone_number",
			queryValue:    "0812340001",
			expectedTotal: 1,
			expectedHN:    "HN-CHK-001",
		},
		{
			name:          "single criterion: email only",
			queryKey:      "email",
			queryValue:    "alpha.chk@hospital.example.com",
			expectedTotal: 1,
			expectedHN:    "HN-CHK-001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := fmt.Sprintf("/patient/search?%s=%s", tt.queryKey, url.QueryEscape(tt.queryValue))
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, endpoint, nil)
			req.AddCookie(cookie1)

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
			}

			var resp response.Response[service.PatientSearchResult]
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp.Status != "success" {
				t.Errorf("expected status 'success', got %s", resp.Status)
			}
			if resp.Data.Total != tt.expectedTotal {
				t.Errorf("expected total %d, got %d", tt.expectedTotal, resp.Data.Total)
			}
			if tt.expectedHN != "" {
				if len(resp.Data.Patients) == 0 {
					t.Fatalf("expected patient %s, got 0 patients in list", tt.expectedHN)
				}
				if resp.Data.Patients[0].PatientHN != tt.expectedHN {
					t.Errorf("expected patient HN %s, got %s", tt.expectedHN, resp.Data.Patients[0].PatientHN)
				}
			}
		})
	}
}

// ============================================================================
// 2. Table-Driven Tests: Multiple criteria combinations
// ============================================================================
func TestPatientSearch_TableDriven_MultipleCriteriaCombinations(t *testing.T) {
	router, cookie1, _, _, _, teardown := setupChecklistFixtures(t)
	defer teardown()

	tests := []struct {
		name          string
		params        map[string]string
		expectedTotal int
		expectedHN    string
	}{
		{
			name: "2 criteria match: first_name + last_name",
			params: map[string]string{
				"first_name": "CheckAlpha",
				"last_name":  "CheckCharoenying",
			},
			expectedTotal: 1,
			expectedHN:    "HN-CHK-001",
		},
		{
			name: "3 criteria match: first_name + date_of_birth + phone_number",
			params: map[string]string{
				"first_name":    "CheckAlpha",
				"date_of_birth": "1973-03-21",
				"phone_number":  "0812340001",
			},
			expectedTotal: 1,
			expectedHN:    "HN-CHK-001",
		},
		{
			name: "4 criteria match: national_id + first_name + last_name + email",
			params: map[string]string{
				"national_id": "9900112233001",
				"first_name":  "CheckAlpha",
				"last_name":   "CheckCharoenying",
				"email":       "alpha.chk@hospital.example.com",
			},
			expectedTotal: 1,
			expectedHN:    "HN-CHK-001",
		},
		{
			name: "all 8 criteria matching simultaneously",
			params: map[string]string{
				"national_id":   "9900112233001",
				"passport_id":   "CHKPASS001",
				"first_name":    "CheckAlpha",
				"middle_name":   "CheckRungruang",
				"last_name":     "CheckCharoenying",
				"date_of_birth": "1973-03-21",
				"phone_number":  "0812340001",
				"email":         "alpha.chk@hospital.example.com",
			},
			expectedTotal: 1,
			expectedHN:    "HN-CHK-001",
		},
		{
			name: "mismatch AND logic: first_name matches but last_name does not",
			params: map[string]string{
				"first_name": "CheckAlpha",
				"last_name":  "WrongLastName",
			},
			expectedTotal: 0,
		},
		{
			name: "mismatch AND logic: national_id matches but phone_number does not",
			params: map[string]string{
				"national_id":  "9900112233001",
				"phone_number": "0899999999",
			},
			expectedTotal: 0,
		},
		{
			name: "mismatch AND logic: national_id matches but date_of_birth does not",
			params: map[string]string{
				"national_id":   "9900112233001",
				"date_of_birth": "1999-12-31",
			},
			expectedTotal: 0,
		},
		{
			name: "mismatch AND logic: phone matches but email does not",
			params: map[string]string{
				"phone_number": "0812340001",
				"email":        "unmatched@hospital.example.com",
			},
			expectedTotal: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := url.Values{}
			for k, v := range tt.params {
				values.Set(k, v)
			}

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/patient/search?"+values.Encode(), nil)
			req.AddCookie(cookie1)

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
			}

			var resp response.Response[service.PatientSearchResult]
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp.Data.Total != tt.expectedTotal {
				t.Errorf("expected total %d, got %d", tt.expectedTotal, resp.Data.Total)
			}
			if tt.expectedHN != "" {
				if len(resp.Data.Patients) == 0 || resp.Data.Patients[0].PatientHN != tt.expectedHN {
					t.Errorf("expected patient %s, got %+v", tt.expectedHN, resp.Data.Patients)
				}
			}
		})
	}
}

// ============================================================================
// 3. Table-Driven Tests: Partial name matching
// ============================================================================
func TestPatientSearch_TableDriven_PartialNameMatching(t *testing.T) {
	router, cookie1, _, _, _, teardown := setupChecklistFixtures(t)
	defer teardown()

	tests := []struct {
		name          string
		queryField    string
		searchTerm    string
		expectedTotal int
		expectedHN    string
	}{
		{
			name:          "English prefix match on first_name (CheckAl -> CheckAlpha)",
			queryField:    "first_name",
			searchTerm:    "CheckAl",
			expectedTotal: 1,
			expectedHN:    "HN-CHK-001",
		},
		{
			name:          "English suffix match on first_name (Alpha -> CheckAlpha)",
			queryField:    "first_name",
			searchTerm:    "Alpha",
			expectedTotal: 1,
			expectedHN:    "HN-CHK-001",
		},
		{
			name:          "English infix match on first_name (kAlp -> CheckAlpha)",
			queryField:    "first_name",
			searchTerm:    "kAlp",
			expectedTotal: 1,
			expectedHN:    "HN-CHK-001",
		},
		{
			name:          "English common prefix match across both patients (Check -> CheckAlpha, CheckBeta)",
			queryField:    "first_name",
			searchTerm:    "Check",
			expectedTotal: 2,
		},
		{
			name:          "Thai prefix match on first_name (เช็กเกอร์กิ -> เช็กเกอร์กิตติ)",
			queryField:    "first_name",
			searchTerm:    "เช็กเกอร์กิ",
			expectedTotal: 1,
			expectedHN:    "HN-CHK-001",
		},
		{
			name:          "Thai suffix match on first_name (พานิช -> เช็กเกอร์พานิช)",
			queryField:    "first_name",
			searchTerm:    "พานิช",
			expectedTotal: 1,
			expectedHN:    "HN-CHK-002",
		},
		{
			name:          "Thai infix match shared across both patients (เช็กเกอร์ -> เช็กเกอร์กิตติ, เช็กเกอร์พานิช)",
			queryField:    "first_name",
			searchTerm:    "เช็กเกอร์",
			expectedTotal: 2,
		},
		{
			name:          "Middle name partial match English prefix (Rung -> CheckRungruang)",
			queryField:    "middle_name",
			searchTerm:    "Rung",
			expectedTotal: 1,
			expectedHN:    "HN-CHK-001",
		},
		{
			name:          "Middle name partial match Thai suffix (เรือง -> เช็กเกอร์รุ่งเรือง)",
			queryField:    "middle_name",
			searchTerm:    "เรือง",
			expectedTotal: 1,
			expectedHN:    "HN-CHK-001",
		},
		{
			name:          "Last name partial match English infix (roenying -> CheckCharoenying)",
			queryField:    "last_name",
			searchTerm:    "roenying",
			expectedTotal: 2,
		},
		{
			name:          "Last name partial match Thai suffix (ยิ่ง -> เช็กเกอร์เจริญยิ่ง)",
			queryField:    "last_name",
			searchTerm:    "ยิ่ง",
			expectedTotal: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := fmt.Sprintf("/patient/search?%s=%s", tt.queryField, url.QueryEscape(tt.searchTerm))
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, endpoint, nil)
			req.AddCookie(cookie1)

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
			}

			var resp response.Response[service.PatientSearchResult]
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp.Data.Total != tt.expectedTotal {
				t.Errorf("expected total %d for term %q, got %d", tt.expectedTotal, tt.searchTerm, resp.Data.Total)
			}
			if tt.expectedHN != "" {
				if len(resp.Data.Patients) == 0 || resp.Data.Patients[0].PatientHN != tt.expectedHN {
					t.Errorf("expected HN %s, got %+v", tt.expectedHN, resp.Data.Patients)
				}
			}
		})
	}
}

// ============================================================================
// 4. Table-Driven Tests: Case-insensitive search
// ============================================================================
func TestPatientSearch_TableDriven_CaseInsensitiveSearch(t *testing.T) {
	router, cookie1, _, _, _, teardown := setupChecklistFixtures(t)
	defer teardown()

	tests := []struct {
		name          string
		queryField    string
		searchTerm    string
		expectedTotal int
		expectedHN    string
	}{
		{
			name:          "All uppercase first_name (CHECKALPHA)",
			queryField:    "first_name",
			searchTerm:    "CHECKALPHA",
			expectedTotal: 1,
			expectedHN:    "HN-CHK-001",
		},
		{
			name:          "All lowercase first_name (checkalpha)",
			queryField:    "first_name",
			searchTerm:    "checkalpha",
			expectedTotal: 1,
			expectedHN:    "HN-CHK-001",
		},
		{
			name:          "Mixed / Camel case first_name (cHeCkAlPhA)",
			queryField:    "first_name",
			searchTerm:    "cHeCkAlPhA",
			expectedTotal: 1,
			expectedHN:    "HN-CHK-001",
		},
		{
			name:          "All uppercase last_name (CHECKCHAROENYING)",
			queryField:    "last_name",
			searchTerm:    "CHECKCHAROENYING",
			expectedTotal: 2,
		},
		{
			name:          "All lowercase last_name (checkcharoenying)",
			queryField:    "last_name",
			searchTerm:    "checkcharoenying",
			expectedTotal: 2,
		},
		{
			name:          "Lowercase passport_id (chkpass001 vs stored CHKPASS001)",
			queryField:    "passport_id",
			searchTerm:    "chkpass001",
			expectedTotal: 1,
			expectedHN:    "HN-CHK-001",
		},
		{
			name:          "Mixed case passport_id (ChkPass001)",
			queryField:    "passport_id",
			searchTerm:    "ChkPass001",
			expectedTotal: 1,
			expectedHN:    "HN-CHK-001",
		},
		{
			name:          "Uppercase email (ALPHA.CHK@HOSPITAL.EXAMPLE.COM)",
			queryField:    "email",
			searchTerm:    "ALPHA.CHK@HOSPITAL.EXAMPLE.COM",
			expectedTotal: 1,
			expectedHN:    "HN-CHK-001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := fmt.Sprintf("/patient/search?%s=%s", tt.queryField, url.QueryEscape(tt.searchTerm))
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, endpoint, nil)
			req.AddCookie(cookie1)

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
			}

			var resp response.Response[service.PatientSearchResult]
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp.Data.Total != tt.expectedTotal {
				t.Errorf("expected total %d for case test %q, got %d", tt.expectedTotal, tt.searchTerm, resp.Data.Total)
			}
			if tt.expectedHN != "" {
				if len(resp.Data.Patients) == 0 || resp.Data.Patients[0].PatientHN != tt.expectedHN {
					t.Errorf("expected HN %s, got %+v", tt.expectedHN, resp.Data.Patients)
				}
			}
		})
	}
}

// ============================================================================
// 5. Table-Driven Tests: Empty results
// ============================================================================
func TestPatientSearch_TableDriven_EmptyResults(t *testing.T) {
	router, cookie1, _, _, _, teardown := setupChecklistFixtures(t)
	defer teardown()

	tests := []struct {
		name   string
		params map[string]string
	}{
		{
			name:   "non-existent national ID returns 0 results",
			params: map[string]string{"national_id": "0000000000000"},
		},
		{
			name:   "non-existent passport ID returns 0 results",
			params: map[string]string{"passport_id": "NOTFOUND999"},
		},
		{
			name:   "non-existent first name returns 0 results",
			params: map[string]string{"first_name": "NonExistentNameXYZ"},
		},
		{
			name:   "non-existent last name returns 0 results",
			params: map[string]string{"last_name": "ZeroMatchSurname"},
		},
		{
			name:   "non-existent date of birth returns 0 results",
			params: map[string]string{"date_of_birth": "1901-01-01"},
		},
		{
			name:   "non-existent phone number returns 0 results",
			params: map[string]string{"phone_number": "0000000000"},
		},
		{
			name:   "non-existent email address returns 0 results",
			params: map[string]string{"email": "unregistered@nowhere.example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := url.Values{}
			for k, v := range tt.params {
				values.Set(k, v)
			}

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/patient/search?"+values.Encode(), nil)
			req.AddCookie(cookie1)

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
			}

			var resp response.Response[service.PatientSearchResult]
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp.Status != "success" {
				t.Errorf("expected status 'success', got %s", resp.Status)
			}
			if resp.Data.Total != 0 {
				t.Errorf("expected total 0, got %d", resp.Data.Total)
			}
			if len(resp.Data.Patients) != 0 {
				t.Errorf("expected empty patients array, got len=%d", len(resp.Data.Patients))
			}
			// Verify non-null JSON array in payload
			if resp.Data.Patients == nil {
				t.Error("expected non-null empty array [] for patients, got nil")
			}
		})
	}
}

// ============================================================================
// ============================================================================
// 6. Table-Driven Tests: Unpaginated search and ignored query params
// ============================================================================
func TestPatientSearch_TableDriven_PaginationEdgeCases(t *testing.T) {
	router, cookie1, _, _, _, teardown := setupChecklistFixtures(t)
	defer teardown()

	tests := []struct {
		name           string
		params         map[string]string
		expectedStatus int
		expectedTotal  int
		expectedCount  int
	}{
		{
			name: "Returns all matching patients without pagination",
			params: map[string]string{
				"last_name": "CheckCharoenying",
			},
			expectedStatus: http.StatusOK,
			expectedTotal:  2,
			expectedCount:  2,
		},
		{
			name: "Ignored limit query parameter returns all items",
			params: map[string]string{
				"last_name": "CheckCharoenying",
				"limit":     "1",
			},
			expectedStatus: http.StatusOK,
			expectedTotal:  2,
			expectedCount:  2,
		},
		{
			name: "Ignored offset query parameter returns all items",
			params: map[string]string{
				"last_name": "CheckCharoenying",
				"offset":    "10",
			},
			expectedStatus: http.StatusOK,
			expectedTotal:  2,
			expectedCount:  2,
		},
		{
			name: "Ignored limit and offset query parameters combined",
			params: map[string]string{
				"last_name": "CheckCharoenying",
				"limit":     "1",
				"offset":    "1",
			},
			expectedStatus: http.StatusOK,
			expectedTotal:  2,
			expectedCount:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := url.Values{}
			for k, v := range tt.params {
				values.Set(k, v)
			}

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/patient/search?"+values.Encode(), nil)
			req.AddCookie(cookie1)

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectedStatus == http.StatusOK {
				var resp response.Response[service.PatientSearchResult]
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if resp.Data.Total != tt.expectedTotal {
					t.Errorf("expected total %d, got %d", tt.expectedTotal, resp.Data.Total)
				}
				if len(resp.Data.Patients) != tt.expectedCount {
					t.Errorf("expected patient count %d, got %d", tt.expectedCount, len(resp.Data.Patients))
				}
			} else {
				// Verify RFC 7807 Problem Details
				var prob middleware.ProblemDetails
				if err := json.Unmarshal(w.Body.Bytes(), &prob); err != nil {
					t.Fatalf("failed to decode RFC 7807 problem: %v", err)
				}
				if prob.Status != tt.expectedStatus {
					t.Errorf("expected problem status %d, got %d", tt.expectedStatus, prob.Status)
				}
			}
		})
	}
}

// ============================================================================
// 7. Table-Driven Tests: Special characters in search terms
// ============================================================================
func TestPatientSearch_TableDriven_SpecialCharacters(t *testing.T) {
	router, cookie1, _, _, _, teardown := setupChecklistFixtures(t)
	defer teardown()

	tests := []struct {
		name          string
		queryField    string
		searchTerm    string
		expectedTotal int
		description   string
	}{
		{
			name:          "Percent literal '%' must NOT match all patients as wildcard",
			queryField:    "first_name",
			searchTerm:    "%",
			expectedTotal: 0,
			description:   "Escaped literal % should not match patients lacking literal %",
		},
		{
			name:          "Underscore literal '_' must NOT match single character wildcard",
			queryField:    "first_name",
			searchTerm:    "CheckAl_ha",
			expectedTotal: 0,
			description:   "Escaped literal _ should not match CheckAlpha",
		},
		{
			name:          "Backslash escape character '\\'",
			queryField:    "first_name",
			searchTerm:    "Check\\Alpha",
			expectedTotal: 0,
			description:   "Escaped backslash handled safely",
		},
		{
			name:          "Single quote in name (O'Connor style)",
			queryField:    "first_name",
			searchTerm:    "Check'Alpha",
			expectedTotal: 0,
			description:   "Single quote treated as literal string",
		},
		{
			name:          "Double quote in search string",
			queryField:    "first_name",
			searchTerm:    `"CheckAlpha"`,
			expectedTotal: 0,
			description:   "Double quotes treated as literal string",
		},
		{
			name:          "Hyphens in search string",
			queryField:    "phone_number",
			searchTerm:    "081-234-0001",
			expectedTotal: 0,
			description:   "Hyphens match literal phone number",
		},
		{
			name:          "Thai vowel and tone marks (สระอิ, ไม้ไต่คู้)",
			queryField:    "first_name",
			searchTerm:    "เกอร์กิตติ",
			expectedTotal: 1, // Matches เช็กเกอร์กิตติ (HN-CHK-001)
			description:   "Valid Thai diacritics and vowel substring matching",
		},
		{
			name:          "Parentheses in query",
			queryField:    "first_name",
			searchTerm:    "(CheckAlpha)",
			expectedTotal: 0,
			description:   "Parentheses treated as literal characters",
		},
		{
			name:          "Multiple wildcards sequence (%_%_%)",
			queryField:    "first_name",
			searchTerm:    "%_%_%",
			expectedTotal: 0,
			description:   "Combinations of LIKE wildcards safely escaped",
		},
		{
			name:          "Percent wildcard in passport_id",
			queryField:    "passport_id",
			searchTerm:    "%",
			expectedTotal: 0,
			description:   "Passport ID with % does not match all passports",
		},
		{
			name:          "Percent wildcard in email",
			queryField:    "email",
			searchTerm:    "%@%",
			expectedTotal: 0,
			description:   "Email with % does not match all emails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := fmt.Sprintf("/patient/search?%s=%s", tt.queryField, url.QueryEscape(tt.searchTerm))
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, endpoint, nil)
			req.AddCookie(cookie1)

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
			}

			var resp response.Response[service.PatientSearchResult]
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp.Data.Total != tt.expectedTotal {
				t.Errorf("expected total %d for special char query %q, got %d (%s)",
					tt.expectedTotal, tt.searchTerm, resp.Data.Total, tt.description)
			}
		})
	}
}

// ============================================================================
// 8. Table-Driven Tests: Cross-hospital isolation
// ============================================================================
func TestPatientSearch_TableDriven_CrossHospitalIsolation(t *testing.T) {
	router, cookie1, cookie2, _, _, teardown := setupChecklistFixtures(t)
	defer teardown()

	tests := []struct {
		name          string
		actingCookie  *http.Cookie
		params        map[string]string
		expectedTotal int
		expectedHN    string
		forbiddenHN   string
	}{
		{
			name:         "Staff 1 (HOSP001) searches duplicate name: sees only Patient A (HOSP001), never Patient C (HOSP002)",
			actingCookie: cookie1,
			params: map[string]string{
				"first_name": "CheckAlpha",
				"last_name":  "CheckCharoenying",
			},
			expectedTotal: 1,
			expectedHN:    "HN-CHK-001",
			forbiddenHN:   "HN-CHK-003",
		},
		{
			name:         "Staff 2 (HOSP002) searches duplicate name: sees only Patient C (HOSP002), never Patient A (HOSP001)",
			actingCookie: cookie2,
			params: map[string]string{
				"first_name": "CheckAlpha",
				"last_name":  "CheckCharoenying",
			},
			expectedTotal: 1,
			expectedHN:    "HN-CHK-003",
			forbiddenHN:   "HN-CHK-001",
		},
		{
			name:         "Staff 1 (HOSP001) searches by National ID of Patient C (in HOSP002): returns 0 results",
			actingCookie: cookie1,
			params: map[string]string{
				"national_id": "9900112233003",
			},
			expectedTotal: 0,
			expectedHN:    "",
			forbiddenHN:   "HN-CHK-003",
		},
		{
			name:         "Staff 2 (HOSP002) searches by National ID of Patient A (in HOSP001): returns 0 results",
			actingCookie: cookie2,
			params: map[string]string{
				"national_id": "9900112233001",
			},
			expectedTotal: 0,
			expectedHN:    "",
			forbiddenHN:   "HN-CHK-001",
		},
		{
			name:         "Staff 2 (HOSP002) searches by Patient B's phone (in HOSP001): returns 0 results",
			actingCookie: cookie2,
			params: map[string]string{
				"phone_number": "0812340002",
			},
			expectedTotal: 0,
			expectedHN:    "",
			forbiddenHN:   "HN-CHK-002",
		},
		{
			name:         "Staff 1 (HOSP001) searches by Patient C's email (in HOSP002): returns 0 results",
			actingCookie: cookie1,
			params: map[string]string{
				"email": "alpha.chula@hospital.example.com",
			},
			expectedTotal: 0,
			expectedHN:    "",
			forbiddenHN:   "HN-CHK-003",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := url.Values{}
			for k, v := range tt.params {
				values.Set(k, v)
			}

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/patient/search?"+values.Encode(), nil)
			req.AddCookie(tt.actingCookie)

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
			}

			var resp response.Response[service.PatientSearchResult]
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp.Data.Total != tt.expectedTotal {
				t.Errorf("expected total %d, got %d", tt.expectedTotal, resp.Data.Total)
			}

			for _, p := range resp.Data.Patients {
				if tt.forbiddenHN != "" && p.PatientHN == tt.forbiddenHN {
					t.Fatalf("CRITICAL ISOLATION BREACH: found forbidden patient %s belonging to another hospital", tt.forbiddenHN)
				}
			}

			if tt.expectedHN != "" {
				if len(resp.Data.Patients) == 0 || resp.Data.Patients[0].PatientHN != tt.expectedHN {
					t.Errorf("expected patient %s, got %+v", tt.expectedHN, resp.Data.Patients)
				}
			}
		})
	}
}

// ============================================================================
// Security Tests: Auth enforcement
// ============================================================================
func TestPatientSearch_Security_AuthEnforcement(t *testing.T) {
	router, cookie1, _, _, _, teardown := setupChecklistFixtures(t)
	defer teardown()

	cfg := config.Load()
	jwtService := auth.NewJWTService(cfg.JWT.Secret, time.Duration(cfg.JWT.ExpirySeconds)*time.Second)

	// Create tokens for edge cases
	expiredJWTService := auth.NewJWTService(cfg.JWT.Secret, -1*time.Hour)
	expiredToken, _ := expiredJWTService.GenerateToken("staff-1", "hosp-1", "doctor")

	tamperedKeyService := auth.NewJWTService("completely-different-wrong-secret-key-1234567890", 1*time.Hour)
	tamperedSignatureToken, _ := tamperedKeyService.GenerateToken("staff-1", "hosp-1", "doctor")

	noHospitalToken, _ := jwtService.GenerateToken("staff-1", "", "doctor")
	noStaffIDToken, _ := jwtService.GenerateToken("", "hosp-1", "doctor")

	tests := []struct {
		name           string
		cookieValue    string
		omitCookie     bool
		expectedStatus int
		expectRFC7807  bool
	}{
		{
			name:           "Missing cookie completely returns 401",
			omitCookie:     true,
			expectedStatus: http.StatusUnauthorized,
			expectRFC7807:  true,
		},
		{
			name:           "Empty cookie value returns 401",
			cookieValue:    "",
			expectedStatus: http.StatusUnauthorized,
			expectRFC7807:  true,
		},
		{
			name:           "Garbage malformed token string returns 401",
			cookieValue:    "not.a.valid.jwt.token",
			expectedStatus: http.StatusUnauthorized,
			expectRFC7807:  true,
		},
		{
			name:           "Expired JWT token returns 401",
			cookieValue:    expiredToken,
			expectedStatus: http.StatusUnauthorized,
			expectRFC7807:  true,
		},
		{
			name:           "Tampered signature (signed with different secret) returns 401",
			cookieValue:    tamperedSignatureToken,
			expectedStatus: http.StatusUnauthorized,
			expectRFC7807:  true,
		},
		{
			name:           "JWT missing hospital_id claim returns 401",
			cookieValue:    noHospitalToken,
			expectedStatus: http.StatusUnauthorized,
			expectRFC7807:  true,
		},
		{
			name:           "JWT missing staff_id claim returns 401",
			cookieValue:    noStaffIDToken,
			expectedStatus: http.StatusUnauthorized,
			expectRFC7807:  true,
		},
		{
			name:           "Valid session token returns 200",
			cookieValue:    cookie1.Value,
			expectedStatus: http.StatusOK,
			expectRFC7807:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/patient/search?first_name=CheckAlpha", nil)

			if !tt.omitCookie {
				req.AddCookie(&http.Cookie{
					Name:  middleware.SessionCookieName,
					Value: tt.cookieValue,
				})
			}

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectRFC7807 {
				if ct := w.Header().Get("Content-Type"); ct != "application/problem+json" {
					t.Errorf("expected Content-Type application/problem+json, got %q", ct)
				}
				var prob middleware.ProblemDetails
				if err := json.Unmarshal(w.Body.Bytes(), &prob); err != nil {
					t.Fatalf("failed to decode problem details: %v", err)
				}
				if prob.Status != tt.expectedStatus {
					t.Errorf("expected problem status %d, got %d", tt.expectedStatus, prob.Status)
				}
			}
		})
	}
}

// ============================================================================
// Security Tests: Hospital scoping
// ============================================================================
func TestPatientSearch_Security_HospitalScoping(t *testing.T) {
	router, cookie1, _, _, hosp2, teardown := setupChecklistFixtures(t)
	defer teardown()

	tests := []struct {
		name         string
		method       string
		url          string
		body         string
		headers      map[string]string
		validateDesc string
	}{
		{
			name:   "Query parameter tampering: client sends ?hospital_id=<HOSP002_UUID>",
			method: http.MethodPost,
			url:    fmt.Sprintf("/patient/search?first_name=CheckAlpha&hospital_id=%s", hosp2.ID),
			body:   "",
			validateDesc: "Must NOT switch hospital scope to HOSP002; only HOSP001 records returned",
		},
		{
			name:   "JSON body tampering: client sends {\"hospital_id\": \"<HOSP002_UUID>\"}",
			method: http.MethodPost,
			url:    "/patient/search",
			body:   fmt.Sprintf(`{"first_name":"CheckAlpha","hospital_id":%q}`, hosp2.ID),
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			validateDesc: "Must NOT switch hospital scope to HOSP002 via JSON body; only HOSP001 records returned",
		},
		{
			name:   "Header injection: client sends X-Hospital-ID or Hospital-ID header",
			method: http.MethodPost,
			url:    "/patient/search?first_name=CheckAlpha",
			body:   "",
			headers: map[string]string{
				"X-Hospital-ID": hosp2.ID,
			},
			validateDesc: "Must ignore spoofed hospital headers; strictly enforce JWT claim",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyReader *bytes.Reader
			if tt.body != "" {
				bodyReader = bytes.NewReader([]byte(tt.body))
			} else {
				bodyReader = bytes.NewReader(nil)
			}

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(tt.method, tt.url, bodyReader)
			req.AddCookie(cookie1)

			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
			}

			var resp response.Response[service.PatientSearchResult]
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			// Verify that results ONLY belong to hosp1, never hosp2
			for _, p := range resp.Data.Patients {
				if p.PatientHN != "HN-CHK-001" && p.PatientHN != "HN-CHK-002" {
					t.Fatalf("CRITICAL SECURITY VIOLATION (%s): returned patient %s not belonging to hosp1",
						tt.validateDesc, p.PatientHN)
				}
				if p.PatientHN == "HN-CHK-003" {
					t.Fatalf("CRITICAL SECURITY VIOLATION: leaked patient HN-CHK-003 from HOSP002!")
				}
			}
		})
	}
}

// ============================================================================
// Security Tests: SQL injection prevention
// ============================================================================
func TestPatientSearch_Security_SQLInjectionPrevention(t *testing.T) {
	router, cookie1, _, _, _, teardown := setupChecklistFixtures(t)
	defer teardown()

	tests := []struct {
		name        string
		params      map[string]string
		jsonBody    string
		description string
	}{
		{
			name: "Tautology injection in first_name: ' OR '1'='1",
			params: map[string]string{
				"first_name": "' OR '1'='1",
			},
			description: "Classic tautology must be safely parameterized and ILIKE-escaped",
		},
		{
			name: "Tautology injection with comments in last_name: ' OR 1=1 --",
			params: map[string]string{
				"last_name": "' OR 1=1 --",
			},
			description: "Trailing comment must not truncate query condition",
		},
		{
			name: "Stacked query injection in national_id: 9900112233001'; DROP TABLE patient; --",
			params: map[string]string{
				"national_id": "9900112233001'; DROP TABLE patient; --",
			},
			description: "Stacked statement must not execute",
		},
		{
			name: "UNION SELECT injection in passport_id",
			params: map[string]string{
				"passport_id": "' UNION SELECT gen_random_uuid(), gen_random_uuid(), 'HACK', 'HACK' --",
			},
			description: "UNION injection must be treated as literal passport search",
		},
		{
			name: "Comment injection in email: admin'--@example.com",
			params: map[string]string{
				"email": "admin'--@example.com",
			},
			description: "SQL comments in email must be safely parameterized",
		},
		{
			name: "Time-based blind SQL injection: '; SELECT pg_sleep(3); --",
			params: map[string]string{
				"first_name": "'; SELECT pg_sleep(3); --",
			},
			description: "Sleep injection must not halt or delay query execution",
		},
		{
			name: "Subquery injection in phone_number",
			params: map[string]string{
				"phone_number": "(SELECT hospital_id FROM hospital LIMIT 1)",
			},
			description: "Subquery syntax must not execute",
		},
		{
			name:     "JSON body SQL injection payload: {\"first_name\": \"' OR ''='\"}",
			jsonBody: `{"first_name": "' OR ''='"}`,
			description: "JSON body search values must be safely parameterized",
		},
		{
			name:     "JSON body stacked query: {\"last_name\": \"'; DELETE FROM staff; --\"}",
			jsonBody: `{"last_name": "'; DELETE FROM staff; --"}`,
			description: "JSON body stacked statements must not execute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			var req *http.Request

			start := time.Now()
			if tt.jsonBody != "" {
				req, _ = http.NewRequest(http.MethodPost, "/patient/search", bytes.NewReader([]byte(tt.jsonBody)))
				req.Header.Set("Content-Type", "application/json")
			} else {
				values := url.Values{}
				for k, v := range tt.params {
					values.Set(k, v)
				}
				req, _ = http.NewRequest(http.MethodPost, "/patient/search?"+values.Encode(), nil)
			}
			req.AddCookie(cookie1)

			router.ServeHTTP(w, req)
			elapsed := time.Since(start)

			// Query must return HTTP 200 (safe 0 results) and not HTTP 500 (syntax error)
			if w.Code != http.StatusOK {
				t.Fatalf("expected HTTP 200, got status %d (%s). Body: %s", w.Code, tt.description, w.Body.String())
			}

			// Time-based blind injection check: elapsed should be well under 1 second
			if elapsed >= 2*time.Second {
				t.Fatalf("possible time-based blind SQL injection! Query took %v", elapsed)
			}

			var resp response.Response[service.PatientSearchResult]
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			// Injected strings should return 0 results since no patient matches the injection literal
			if resp.Data.Total != 0 {
				t.Errorf("expected 0 results for injection payload %s, got %d", tt.description, resp.Data.Total)
			}
		})
	}
}

// ============================================================================
// Edge & Performance Tests: NilDB and Response Time < 200ms
// ============================================================================
func TestPatientSearch_NilDB(t *testing.T) {
	cfg := config.Load()
	router := setupRouter(nil, cfg)

	jwtService := auth.NewJWTService(cfg.JWT.Secret, time.Duration(cfg.JWT.ExpirySeconds)*time.Second)
	token, _ := jwtService.GenerateToken("staff-1", "hosp-1", "doctor")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/patient/search?first_name=CheckAlpha", nil)
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: token})
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500 when db is nil, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestPatientSearch_Performance_ResponseTimeUnder200ms(t *testing.T) {
	router, cookie1, _, _, _, teardown := setupChecklistFixtures(t)
	defer teardown()

	const iterations = 10
	var totalDuration time.Duration

	for i := 0; i < iterations; i++ {
		wPerf := httptest.NewRecorder()
		reqPerf, _ := http.NewRequest(http.MethodPost, "/patient/search?first_name=CheckAlpha", nil)
		reqPerf.AddCookie(cookie1)

		start := time.Now()
		router.ServeHTTP(wPerf, reqPerf)
		elapsed := time.Since(start)

		totalDuration += elapsed
		if elapsed >= 200*time.Millisecond {
			t.Errorf("iteration %d: expected response time < 200ms, got %v", i+1, elapsed)
		}
		if wPerf.Code != http.StatusOK {
			t.Errorf("iteration %d: expected status 200, got %d", i+1, wPerf.Code)
		}
	}

	avgDuration := totalDuration / iterations
	t.Logf("POST /patient/search average response time over %d iterations: %v (target: < 200ms)", iterations, avgDuration)
}

func TestPatientSearch_Performance_1000Patients_ResponseTimeUnder200ms(t *testing.T) {
	cfg := config.Load()
	db, err := database.New(cfg.DB)
	if err != nil {
		t.Skipf("skipping test: database connection not available: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	router := setupRouter(db, cfg)

	hospRepo := repository.NewHospitalRepository(db)
	hosp1, err := hospRepo.FindByHN(ctx, "HOSP001")
	if err != nil || hosp1 == nil {
		t.Skipf("skipping test: hospital HOSP001 not found: %v", err)
	}

	jwtService := auth.NewJWTService(cfg.JWT.Secret, time.Duration(cfg.JWT.ExpirySeconds)*time.Second)
	token, _ := jwtService.GenerateToken("perf-staff", hosp1.ID, "doctor")

	const totalPatients = 1000
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("failed to start transaction: %v", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO patient (
			hospital_id, patient_hn, national_id,
			first_name_th, last_name_th, first_name_en, last_name_en,
			date_of_birth, phone_number, gender
		) VALUES (
			$1, $2, $3,
			'ทดสอบ', 'เพอร์ฟ', 'PerfFirst', 'PerfCheckLast',
			'1990-01-01', '0810000000', 'M'
		)
	`)
	if err != nil {
		t.Fatalf("failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	for i := 1; i <= totalPatients; i++ {
		hn := fmt.Sprintf("HN-PERFCHK-%04d", i)
		natID := fmt.Sprintf("9100%09d", i)
		if _, err := stmt.ExecContext(ctx, hosp1.ID, hn, natID); err != nil {
			t.Fatalf("failed to insert batch patient %d: %v", i, err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("failed to commit 1000 patients: %v", err)
	}

	defer func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM patient WHERE last_name_en = 'PerfCheckLast' AND hospital_id = $1", hosp1.ID)
	}()

	wQuery := httptest.NewRecorder()
	reqQuery, _ := http.NewRequest(http.MethodPost, "/patient/search?last_name=PerfCheckLast", nil)
	reqQuery.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: token})

	qStart := time.Now()
	router.ServeHTTP(wQuery, reqQuery)
	qElapsed := time.Since(qStart)

	if wQuery.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", wQuery.Code, wQuery.Body.String())
	}

	var resQuery response.Response[service.PatientSearchResult]
	if err := json.Unmarshal(wQuery.Body.Bytes(), &resQuery); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resQuery.Data.Total != totalPatients {
		t.Errorf("expected total count %d, got %d", totalPatients, resQuery.Data.Total)
	}
	if len(resQuery.Data.Patients) != totalPatients {
		t.Errorf("expected patient count %d, got %d", totalPatients, len(resQuery.Data.Patients))
	}
	if qElapsed >= 200*time.Millisecond {
		t.Errorf("expected response time < 200ms with 1000 records, got %v", qElapsed)
	}
	t.Logf("Querying over 1000 records took %v (total matching 1000)", qElapsed)
}

