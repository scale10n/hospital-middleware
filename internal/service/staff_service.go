package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"hospital-middleware/internal/auth"
	"hospital-middleware/internal/models"
	"hospital-middleware/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrHospitalNotFound is returned when the hospital HN does not exist.
	ErrHospitalNotFound = errors.New("hospital not found")
	// ErrStaffAlreadyExists is returned when a staff member with the same username already exists for the hospital.
	ErrStaffAlreadyExists = errors.New("staff already exists")
	// ErrInvalidCredentials is returned when login authentication fails.
	ErrInvalidCredentials = errors.New("invalid username or password")
)

// CreateStaffInput contains the input parameters for creating a staff member.
type CreateStaffInput struct {
	Username string
	Password string
	Hospital string
}

// LoginStaffInput contains the input parameters for staff login.
type LoginStaffInput struct {
	Username string
	Password string
	Hospital string
}

// StaffInfoResponse contains staff profile details returned after creation or login.
type StaffInfoResponse struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	HospitalHN   string    `json:"hospital_hn"`
	HospitalName string    `json:"hospital_name"`
	CreatedAt    time.Time `json:"created_at"`
}

// SessionInfoResponse contains session metadata in the response.
type SessionInfoResponse struct {
	ExpiresAt time.Time `json:"expires_at"`
}

// StaffResponseData contains the payload returned after successful staff creation or login (Option 1).
type StaffResponseData struct {
	Staff   StaffInfoResponse   `json:"staff"`
	Session SessionInfoResponse `json:"session"`
}

// CreateStaffResult contains the staff response data and the generated session token.
type CreateStaffResult struct {
	Staff *StaffResponseData `json:"staff"`
	Token string             `json:"token"`
}

// LoginStaffResult contains the staff response data and the generated session token upon login.
type LoginStaffResult struct {
	Staff *StaffResponseData `json:"staff"`
	Token string             `json:"token"`
}

// StaffService defines the business logic operations for staff management.
type StaffService interface {
	CreateStaff(ctx context.Context, input CreateStaffInput) (*CreateStaffResult, error)
	LoginStaff(ctx context.Context, input LoginStaffInput) (*LoginStaffResult, error)
}

type staffService struct {
	hospitalRepo repository.HospitalRepository
	staffRepo    repository.StaffRepository
	sessionRepo  repository.StaffSessionRepository
	tokenService auth.TokenService
}

// NewStaffService constructs a new StaffService with the given repositories and optional services.
func NewStaffService(hospitalRepo repository.HospitalRepository, staffRepo repository.StaffRepository, extras ...any) StaffService {
	var sessionRepo repository.StaffSessionRepository
	var tokenService auth.TokenService

	for _, extra := range extras {
		switch v := extra.(type) {
		case repository.StaffSessionRepository:
			sessionRepo = v
		case auth.TokenService:
			tokenService = v
		}
	}

	if tokenService == nil {
		tokenService = auth.NewJWTService("hospital-middleware-default-secret-key-32bytes", 24*time.Hour)
	}

	return &staffService{
		hospitalRepo: hospitalRepo,
		staffRepo:    staffRepo,
		sessionRepo:  sessionRepo,
		tokenService: tokenService,
	}
}

// CreateStaff executes the business rules for staff creation:
// 1. Verifies that the hospital HN exists in the system.
// 2. Checks if the staff username is already registered for the hospital.
// 3. Hashes the raw password with bcrypt.
// 4. Persists the staff record into the staff table.
// 5. Generates a signed JWT session token.
// 6. Persists the session into the staff_session table.
// 7. Returns the pure domain DTO CreateStaffResult.
func (s *staffService) CreateStaff(ctx context.Context, input CreateStaffInput) (*CreateStaffResult, error) {
	if s.hospitalRepo == nil {
		return nil, errors.New("hospital repository is not configured")
	}
	if s.staffRepo == nil {
		return nil, errors.New("staff repository is not configured")
	}

	hosp, err := s.hospitalRepo.FindByHN(ctx, input.Hospital)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrHospitalNotFound
		}
		return nil, fmt.Errorf("find hospital: %w", err)
	}
	if hosp == nil {
		return nil, ErrHospitalNotFound
	}

	staffExists, err := s.staffRepo.ExistsByHospitalAndUsername(ctx, hosp.ID, input.Username)
	if err != nil {
		return nil, fmt.Errorf("check staff exists: %w", err)
	}
	if staffExists {
		return nil, ErrStaffAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	newStaff := &models.Staff{
		HospitalID:   hosp.ID,
		Username:     input.Username,
		PasswordHash: string(hashedPassword),
	}

	created, err := s.staffRepo.Create(ctx, newStaff)
	if err != nil {
		return nil, fmt.Errorf("save staff record: %w", err)
	}

	token, err := s.tokenService.GenerateToken(created.ID, created.HospitalID, created.Username)
	if err != nil {
		return nil, fmt.Errorf("generate jwt token: %w", err)
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	if claims, err := s.tokenService.ValidateToken(token); err == nil && claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}

	// Persist session into staff_session table
	if s.sessionRepo != nil {
		session := &models.StaffSession{
			StaffID:   created.ID,
			Token:     token,
			ExpiresAt: expiresAt,
		}
		if _, err := s.sessionRepo.Create(ctx, session); err != nil {
			return nil, fmt.Errorf("save staff session: %w", err)
		}
	}

	return &CreateStaffResult{
		Staff: &StaffResponseData{
			Staff: StaffInfoResponse{
				ID:           created.ID,
				Username:     created.Username,
				HospitalHN:   hosp.HN,
				HospitalName: hosp.Name,
				CreatedAt:    created.CreatedAt,
			},
			Session: SessionInfoResponse{
				ExpiresAt: expiresAt,
			},
		},
		Token: token,
	}, nil
}

// LoginStaff authenticates a staff member using username, password, and hospital HN:
// 1. Verifies that the hospital HN exists in the system.
// 2. Looks up the staff by hospital ID and username.
// 3. Validates the password hash using bcrypt.
// 4. Generates a signed JWT session token.
// 5. Persists the session into the staff_session table.
// 6. Returns the domain DTO LoginStaffResult.
func (s *staffService) LoginStaff(ctx context.Context, input LoginStaffInput) (*LoginStaffResult, error) {
	if s.staffRepo == nil {
		return nil, errors.New("staff repository is not configured")
	}
	if s.hospitalRepo == nil {
		return nil, errors.New("hospital repository is not configured")
	}

	hosp, err := s.hospitalRepo.FindByHN(ctx, input.Hospital)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrHospitalNotFound
		}
		return nil, fmt.Errorf("find hospital by hn: %w", err)
	}
	if hosp == nil {
		return nil, ErrHospitalNotFound
	}

	staff, err := s.staffRepo.FindByHospitalAndUsername(ctx, hosp.ID, input.Username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("find staff by hospital and username: %w", err)
	}
	if staff == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(staff.PasswordHash), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := s.tokenService.GenerateToken(staff.ID, staff.HospitalID, staff.Username)
	if err != nil {
		return nil, fmt.Errorf("generate jwt token: %w", err)
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	if claims, err := s.tokenService.ValidateToken(token); err == nil && claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}

	// Persist session into staff_session table
	if s.sessionRepo != nil {
		session := &models.StaffSession{
			StaffID:   staff.ID,
			Token:     token,
			ExpiresAt: expiresAt,
		}
		if _, err := s.sessionRepo.Create(ctx, session); err != nil {
			return nil, fmt.Errorf("save staff session: %w", err)
		}
	}

	return &LoginStaffResult{
		Staff: &StaffResponseData{
			Staff: StaffInfoResponse{
				ID:           staff.ID,
				Username:     staff.Username,
				HospitalHN:   hosp.HN,
				HospitalName: hosp.Name,
				CreatedAt:    staff.CreatedAt,
			},
			Session: SessionInfoResponse{
				ExpiresAt: expiresAt,
			},
		},
		Token: token,
	}, nil
}
