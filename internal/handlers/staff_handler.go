package handlers

import (
	"database/sql"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"hospital-middleware/internal/repository"
	"hospital-middleware/internal/response"
	"hospital-middleware/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v.RegisterValidation("password", validatePassword)
	}
}

// validatePassword is a custom validator for Gin/go-playground/validator.
func validatePassword(fl validator.FieldLevel) bool {
	return IsValidPassword(fl.Field().String())
}

// IsValidPassword checks if a password complies with the security policy:
// 1. Length between 8 and 32 characters
// 2. Only English letters, numbers, and allowed ASCII special characters (rejects Thai or other non-English characters)
// 3. At least 1 lowercase letter (a-z)
// 4. At least 1 uppercase letter (A-Z)
// 5. At least 1 digit (0-9)
// 6. At least 1 special character ([!@#$%^&*...])
func IsValidPassword(p string) bool {
	length := utf8.RuneCountInString(p)
	if length < 8 || length > 32 {
		return false
	}

	const allowedSpecial = "!@#$%^&*()-_=+[]{}|;:'\",.<>/?`~\\"

	var (
		hasLower   bool
		hasUpper   bool
		hasDigit   bool
		hasSpecial bool
	)

	for _, r := range p {
		switch {
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= '0' && r <= '9':
			hasDigit = true
		case strings.ContainsRune(allowedSpecial, r):
			hasSpecial = true
		default:
			// Rejects any characters not in English lowercase, uppercase, digits, or allowed ASCII special symbols.
			// This specifically rejects Thai characters (e.g. 'ห'), spaces, emojis, or any non-English character.
			return false
		}
	}

	return hasLower && hasUpper && hasDigit && hasSpecial
}

// CreateStaffRequest represents the payload for creating a staff member.
type CreateStaffRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	Password string `json:"password" binding:"required,password"`
	Hospital string `json:"hospital" binding:"required"`
}

// StaffHandler handles staff-related HTTP transport requests.
type StaffHandler struct {
	staffService service.StaffService
}

// NewStaffHandler constructs a StaffHandler. It wires the repository and service layers if a DB is provided.
func NewStaffHandler(db ...*sql.DB) *StaffHandler {
	var s service.StaffService
	if len(db) > 0 && db[0] != nil {
		hospitalRepo := repository.NewHospitalRepository(db[0])
		staffRepo := repository.NewStaffRepository(db[0])
		staffSessionRepo := repository.NewStaffSessionRepository(db[0])
		s = service.NewStaffService(hospitalRepo, staffRepo, staffSessionRepo)
	} else {
		s = service.NewStaffService(nil, nil)
	}
	return &StaffHandler{staffService: s}
}

// NewStaffHandlerWithService constructs a StaffHandler with an explicit StaffService (ideal for DI and mocking).
func NewStaffHandlerWithService(staffService service.StaffService) *StaffHandler {
	return &StaffHandler{staffService: staffService}
}

// RegisterRoutes registers staff endpoints onto the given Gin router group.
func (h *StaffHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/create", h.CreateStaff)
}

// CreateStaff handles POST /staff/create requests:
// Binds request JSON, delegates to StaffService, attaches errors to context, and returns 201 on success.
func (h *StaffHandler) CreateStaff(c *gin.Context) {
	var req CreateStaffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	input := service.CreateStaffInput{
		Username: req.Username,
		Password: req.Password,
		Hospital: req.Hospital,
	}

	output, err := h.staffService.CreateStaff(c.Request.Context(), input)
	if err != nil {
		_ = c.Error(err)
		return
	}

	// Set SameSite mode for CSRF mitigation
	c.SetSameSite(http.SameSiteLaxMode)

	// Set HttpOnly session cookie on the client response
	secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	maxAge := 86400
	if output != nil && output.Staff != nil && !output.Staff.Session.ExpiresAt.IsZero() {
		if sec := int(time.Until(output.Staff.Session.ExpiresAt).Seconds()); sec > 0 {
			maxAge = sec
		}
	}

	c.SetCookie(
		"session_token",
		output.Token,
		maxAge,
		"/",
		"",
		secure,
		true, // HttpOnly: prevents client-side script from accessing the cookie (XSS protection)
	)

	c.JSON(http.StatusCreated, response.Success(output.Staff, "staff created successfully"))
}
