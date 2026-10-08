package auth_test

import (
	"testing"
	"time"

	"hospital-middleware/internal/auth"
)

func TestJWTService_GenerateAndValidate(t *testing.T) {
	secret := "test-secret-key-32bytes-for-hmac"
	jwtSvc := auth.NewJWTService(secret, 1*time.Hour)

	staffID := "staff-uuid-123"
	hospitalID := "hosp-uuid-456"
	username := "admin01"

	tokenStr, err := jwtSvc.GenerateToken(staffID, hospitalID, username)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	if tokenStr == "" {
		t.Fatal("expected non-empty token string")
	}

	claims, err := jwtSvc.ValidateToken(tokenStr)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if claims.StaffID != staffID {
		t.Errorf("expected staff ID %q, got %q", staffID, claims.StaffID)
	}
	if claims.HospitalID != hospitalID {
		t.Errorf("expected hospital ID %q, got %q", hospitalID, claims.HospitalID)
	}
	if claims.Username != username {
		t.Errorf("expected username %q, got %q", username, claims.Username)
	}
}

func TestJWTService_InvalidSecret(t *testing.T) {
	jwtSvc1 := auth.NewJWTService("secret-one-12345678901234567890", 1*time.Hour)
	jwtSvc2 := auth.NewJWTService("secret-two-12345678901234567890", 1*time.Hour)

	tokenStr, err := jwtSvc1.GenerateToken("staff-1", "hosp-1", "user1")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = jwtSvc2.ValidateToken(tokenStr)
	if err == nil {
		t.Fatal("expected error validating token with wrong secret, got nil")
	}
}

func TestJWTService_ExpiredToken(t *testing.T) {
	jwtSvc := auth.NewJWTService("test-secret-key", -1*time.Minute)

	tokenStr, err := jwtSvc.GenerateToken("staff-1", "hosp-1", "user1")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = jwtSvc.ValidateToken(tokenStr)
	if err == nil {
		t.Fatal("expected error validating expired token, got nil")
	}
}
