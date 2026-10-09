package auth

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken is returned when the JWT token signature or claims validation fails.
	ErrInvalidToken = errors.New("invalid or expired jwt token")
)

// StaffClaims represents the payload stored inside the JWT token.
type StaffClaims struct {
	StaffID    string `json:"staff_id"`
	HospitalID string `json:"hospital_id"`
	Username   string `json:"username"`
	jwt.RegisteredClaims
}

// TokenService defines token generation and parsing behavior.
type TokenService interface {
	GenerateToken(staffID, hospitalID, username string) (string, error)
	ValidateToken(tokenString string) (*StaffClaims, error)
}

// JWTService implements TokenService using HMAC-SHA256 (HS256).
type JWTService struct {
	secretKey []byte
	duration  time.Duration
}

// NewJWTService constructs a new JWTService with the given secret and duration.
func NewJWTService(secret string, duration time.Duration) *JWTService {
	if duration == 0 {
		duration = 24 * time.Hour
	}
	return &JWTService{
		secretKey: []byte(secret),
		duration:  duration,
	}
}

// GenerateToken creates and signs a new JWT token for a staff member.
func (j *JWTService) GenerateToken(staffID, hospitalID, username string) (string, error) {
	now := time.Now()
	claims := StaffClaims{
		StaffID:    staffID,
		HospitalID: hospitalID,
		Username:   username,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        strconv.FormatInt(now.UnixNano(), 10),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.duration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(j.secretKey)
	if err != nil {
		return "", fmt.Errorf("sign jwt token: %w", err)
	}

	return signed, nil
}

// ValidateToken parses and verifies the signature and expiration of a JWT token.
func (j *JWTService) ValidateToken(tokenString string) (*StaffClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &StaffClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*StaffClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
