package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"hospital-middleware/internal/middleware"
	"hospital-middleware/internal/repository"
	"hospital-middleware/internal/response"
	"hospital-middleware/internal/service"

	"github.com/gin-gonic/gin"
)

// PatientSearchRequest represents the search payload for searching patients.
// At least one criterion must be provided. Multiple criteria are combined with AND logic.
type PatientSearchRequest struct {
	// 13-digit Thai Citizen Identification Number (เลขบัตรประชาชน 13 หลัก)
	NationalID string `form:"national_id" json:"national_id,omitempty" example:"1100505566779"`
	// Passport number for foreign patients (เลขที่หนังสือเดินทาง)
	PassportID string `form:"passport_id" json:"passport_id,omitempty" example:""`
	// Patient first name in Thai or English (supports case-insensitive partial match)
	FirstName string `form:"first_name" json:"first_name,omitempty" example:""`
	// Patient middle name in Thai or English (supports case-insensitive partial match)
	MiddleName string `form:"middle_name" json:"middle_name,omitempty" example:""`
	// Patient last name in Thai or English (supports case-insensitive partial match)
	LastName string `form:"last_name" json:"last_name,omitempty" example:""`
	// Date of birth formatted as YYYY-MM-DD (วันเกิด รูปแบบ YYYY-MM-DD)
	DateOfBirth string `form:"date_of_birth" json:"date_of_birth,omitempty" example:""`
	// Contact telephone number (exact match)
	PhoneNumber string `form:"phone_number" json:"phone_number,omitempty" example:""`
	// Contact email address (supports case-insensitive partial match)
	Email string `form:"email" json:"email,omitempty" example:""`
}

// PatientSearchOutput represents the response payload for patient search.
type PatientSearchOutput = service.PatientSearchResult

// PatientHandler handles patient-related HTTP requests.
type PatientHandler struct {
	patientService service.PatientService
}

// NewPatientHandler constructs a PatientHandler with an optional PatientService.
func NewPatientHandler(patientService ...service.PatientService) *PatientHandler {
	var s service.PatientService
	if len(patientService) > 0 && patientService[0] != nil {
		s = patientService[0]
	} else {
		s = service.NewPatientService(nil)
	}
	return &PatientHandler{patientService: s}
}

// NewPatientHandlerWithDB constructs a PatientHandler wired to a database instance.
func NewPatientHandlerWithDB(db *sql.DB) *PatientHandler {
	var repo repository.PatientRepository
	if db != nil {
		repo = repository.NewPatientRepository(db)
	}
	return &PatientHandler{patientService: service.NewPatientService(repo)}
}

// SearchPatients handles POST /patient/search requests.
// @Summary      Search patients
// @Description  Search patients within the authenticated staff member's hospital. Multiple criteria are combined with AND logic. At least one search criterion must be provided. Supports partial matching for names and email, exact matching for IDs and phone, and YYYY-MM-DD format for date of birth.
// @Tags         Patient
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Param        request        body     PatientSearchRequest true "Patient search criteria payload"
// @Success      200            {object} response.Response[PatientSearchOutput] "Successful search response with list of matching patients"
// @Failure      400            {object} middleware.ProblemDetails "Bad request: invalid parameters or empty search criteria"
// @Failure      401            {object} middleware.ProblemDetails "Unauthorized: missing or invalid session cookie"
// @Failure      500            {object} middleware.ProblemDetails "Internal server error"
// @Router       /patient/search [post]
func (h *PatientHandler) SearchPatients(c *gin.Context) {
	hospitalID, ok := middleware.GetHospitalID(c)
	if !ok || strings.TrimSpace(hospitalID) == "" {
		_ = c.Error(middleware.NewAppError(http.StatusUnauthorized, "staff hospital affiliation missing"))
		return
	}

	var req PatientSearchRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		_ = c.Error(middleware.NewAppError(http.StatusBadRequest, "invalid query parameters", err))
		return
	}

	// If a JSON request body is present, bind body to also support search params passed via body
	if strings.Contains(c.ContentType(), "application/json") && c.Request.ContentLength > 0 {
		var bodyReq PatientSearchRequest
		if err := c.ShouldBindJSON(&bodyReq); err != nil {
			_ = c.Error(err)
			return
		}

		if req.NationalID == "" {
			req.NationalID = bodyReq.NationalID
		}
		if req.PassportID == "" {
			req.PassportID = bodyReq.PassportID
		}
		if req.FirstName == "" {
			req.FirstName = bodyReq.FirstName
		}
		if req.MiddleName == "" {
			req.MiddleName = bodyReq.MiddleName
		}
		if req.LastName == "" {
			req.LastName = bodyReq.LastName
		}
		if req.DateOfBirth == "" {
			req.DateOfBirth = bodyReq.DateOfBirth
		}
		if req.PhoneNumber == "" {
			req.PhoneNumber = bodyReq.PhoneNumber
		}
		if req.Email == "" {
			req.Email = bodyReq.Email
		}
	}

	input := service.PatientSearchInput{
		NationalID:  req.NationalID,
		PassportID:  req.PassportID,
		FirstName:   req.FirstName,
		MiddleName:  req.MiddleName,
		LastName:    req.LastName,
		DateOfBirth: req.DateOfBirth,
		PhoneNumber: req.PhoneNumber,
		Email:       req.Email,
	}

	result, err := h.patientService.SearchPatients(c.Request.Context(), hospitalID, input)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, response.Success(result, "patients retrieved successfully"))
}

// GetPatientByID handles GET /patient/search/{id} requests.
// @Summary      Search patient by National ID or Passport ID
// @Description  Search and retrieve specific patient details by Thai National ID (13 digits) or Passport ID within the authenticated staff member's hospital.
// @Tags         Patient
// @Produce      json
// @Security     CookieAuth
// @Param        id             path     string true "National ID (13 digits) or Passport ID" example("1100501234567")
// @Success      200            {object} response.Response[service.PatientResponse] "Successful response with patient details"
// @Failure      400            {object} middleware.ProblemDetails "Bad request: invalid national_id or passport_id format"
// @Failure      401            {object} middleware.ProblemDetails "Unauthorized: missing or invalid session cookie"
// @Failure      404            {object} middleware.ProblemDetails "Not found: patient does not exist"
// @Failure      500            {object} middleware.ProblemDetails "Internal server error"
// @Router       /patient/search/{id} [get]
func (h *PatientHandler) GetPatientByID(c *gin.Context) {
	hospitalID, ok := middleware.GetHospitalID(c)
	if !ok || strings.TrimSpace(hospitalID) == "" {
		_ = c.Error(middleware.NewAppError(http.StatusUnauthorized, "staff hospital affiliation missing"))
		return
	}

	id := strings.TrimSpace(c.Param("id"))
	if !service.IsValidIdentity(id) {
		_ = c.Error(service.ErrInvalidPatientID)
		return
	}

	result, err := h.patientService.GetPatientByID(c.Request.Context(), hospitalID, id)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, response.Success(result, "patient retrieved successfully"))
}

