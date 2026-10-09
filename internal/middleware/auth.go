package middleware

import (
	"errors"
	"net/http"
	"strings"

	"hospital-middleware/internal/auth"

	"github.com/gin-gonic/gin"
)

const (
	// SessionCookieName is the cookie name used for staff session JWT token.
	SessionCookieName = "session_token"

	// ContextKeyClaims stores parsed StaffClaims in Gin context.
	ContextKeyClaims = "staff_claims"
	// ContextKeyStaffID stores the staff UUID in Gin context.
	ContextKeyStaffID = "staff_id"
	// ContextKeyHospitalID stores the hospital UUID in Gin context.
	ContextKeyHospitalID = "hospital_id"
	// ContextKeyUsername stores the staff username in Gin context.
	ContextKeyUsername = "username"
)

var (
	// ErrSessionCookieMissing is returned when session_token cookie is absent.
	ErrSessionCookieMissing = errors.New("authentication required: missing session cookie")
	// ErrSessionCookieInvalid is returned when session_token cookie cannot be validated.
	ErrSessionCookieInvalid = errors.New("authentication failed: invalid or expired session token")
	// ErrSessionClaimsInvalid is returned when session claims lack hospital affiliation.
	ErrSessionClaimsInvalid = errors.New("authentication failed: invalid session claims")
)

// CookieAuth validates staff authentication using JWT session cookie (CookieAuth).
func CookieAuth(tokenService auth.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := c.Cookie(SessionCookieName)
		if err != nil || strings.TrimSpace(tokenString) == "" {
			_ = c.Error(NewAppError(http.StatusUnauthorized, ErrSessionCookieMissing.Error(), ErrSessionCookieMissing))
			c.Abort()
			return
		}

		claims, err := tokenService.ValidateToken(tokenString)
		if err != nil {
			_ = c.Error(NewAppError(http.StatusUnauthorized, ErrSessionCookieInvalid.Error(), err))
			c.Abort()
			return
		}

		if claims == nil || strings.TrimSpace(claims.HospitalID) == "" || strings.TrimSpace(claims.StaffID) == "" {
			_ = c.Error(NewAppError(http.StatusUnauthorized, ErrSessionClaimsInvalid.Error(), ErrSessionClaimsInvalid))
			c.Abort()
			return
		}

		c.Set(ContextKeyClaims, claims)
		c.Set(ContextKeyStaffID, claims.StaffID)
		c.Set(ContextKeyHospitalID, claims.HospitalID)
		c.Set(ContextKeyUsername, claims.Username)
		c.Next()
	}
}

// GetStaffClaims extracts authenticated staff claims from Gin context.
func GetStaffClaims(c *gin.Context) (*auth.StaffClaims, bool) {
	val, exists := c.Get(ContextKeyClaims)
	if !exists {
		return nil, false
	}
	claims, ok := val.(*auth.StaffClaims)
	return claims, ok
}

// GetHospitalID extracts the authenticated staff's hospital ID from Gin context.
func GetHospitalID(c *gin.Context) (string, bool) {
	val, exists := c.Get(ContextKeyHospitalID)
	if !exists {
		return "", false
	}
	id, ok := val.(string)
	return id, ok
}

// GetStaffID extracts the authenticated staff ID from Gin context.
func GetStaffID(c *gin.Context) (string, bool) {
	val, exists := c.Get(ContextKeyStaffID)
	if !exists {
		return "", false
	}
	id, ok := val.(string)
	return id, ok
}
