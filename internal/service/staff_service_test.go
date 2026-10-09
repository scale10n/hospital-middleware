package service_test

import (
	"context"
	"errors"
	"testing"

	"hospital-middleware/internal/models"
	"hospital-middleware/internal/service"

	"golang.org/x/crypto/bcrypt"
)

// mockHospitalRepository implements repository.HospitalRepository for testing service logic.
type mockHospitalRepository struct {
	exists   bool
	hospital *models.Hospital
	err      error
}

func (m *mockHospitalRepository) ExistsByHN(ctx context.Context, hn string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.exists, nil
}

func (m *mockHospitalRepository) FindByHN(ctx context.Context, hn string) (*models.Hospital, error) {
	if m.err != nil {
		return nil, m.err
	}
	if !m.exists {
		return nil, nil
	}
	if m.hospital != nil {
		return m.hospital, nil
	}
	return &models.Hospital{
		ID:   "00000000-0000-0000-0000-000000000001",
		HN:   hn,
		Name: "Test Hospital",
	}, nil
}

func (m *mockHospitalRepository) FindByID(ctx context.Context, id string) (*models.Hospital, error) {
	if m.err != nil {
		return nil, m.err
	}
	if !m.exists {
		return nil, nil
	}
	if m.hospital != nil {
		return m.hospital, nil
	}
	return &models.Hospital{
		ID:   id,
		HN:   "HOSP001",
		Name: "Test Hospital",
	}, nil
}

// mockStaffRepository implements repository.StaffRepository for testing service logic.
type mockStaffRepository struct {
	exists                         bool
	existsErr                      error
	createdStaff                   *models.Staff
	createErr                      error
	findByUsernameStaff            *models.Staff
	findByUsernameErr              error
	findByHospitalAndUsernameStaff *models.Staff
	findByHospitalAndUsernameErr   error
}

func (m *mockStaffRepository) ExistsByHospitalAndUsername(ctx context.Context, hospitalID string, username string) (bool, error) {
	if m.existsErr != nil {
		return false, m.existsErr
	}
	return m.exists, nil
}

func (m *mockStaffRepository) FindByHospitalAndUsername(ctx context.Context, hospitalID string, username string) (*models.Staff, error) {
	if m.findByHospitalAndUsernameErr != nil {
		return nil, m.findByHospitalAndUsernameErr
	}
	if m.findByHospitalAndUsernameStaff != nil {
		return m.findByHospitalAndUsernameStaff, nil
	}
	if m.findByUsernameStaff != nil && m.findByUsernameStaff.Username == username {
		return m.findByUsernameStaff, nil
	}
	if m.createdStaff != nil && m.createdStaff.Username == username {
		return m.createdStaff, nil
	}
	return nil, nil
}

func (m *mockStaffRepository) FindByUsername(ctx context.Context, username string) (*models.Staff, error) {
	if m.findByUsernameErr != nil {
		return nil, m.findByUsernameErr
	}
	if m.findByUsernameStaff != nil {
		return m.findByUsernameStaff, nil
	}
	if m.createdStaff != nil && m.createdStaff.Username == username {
		return m.createdStaff, nil
	}
	return nil, nil
}

func (m *mockStaffRepository) Create(ctx context.Context, staff *models.Staff) (*models.Staff, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.createdStaff = staff
	ret := *staff
	ret.ID = "generated-staff-uuid"
	return &ret, nil
}

// mockSessionRepository implements repository.SessionRepository for testing service logic.
type mockSessionRepository struct {
	createdSession *models.StaffSession
	createErr      error
}

func (m *mockSessionRepository) Create(ctx context.Context, session *models.StaffSession) (*models.StaffSession, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.createdSession = session
	ret := *session
	ret.ID = "generated-session-uuid"
	return &ret, nil
}

func (m *mockSessionRepository) FindByToken(ctx context.Context, token string) (*models.StaffSession, error) {
	return m.createdSession, nil
}

func (m *mockSessionRepository) DeleteByToken(ctx context.Context, token string) error {
	return nil
}

func (m *mockSessionRepository) DeleteByStaffID(ctx context.Context, staffID string) error {
	return nil
}

func TestStaffService_CreateStaff_Success(t *testing.T) {
	mockHospRepo := &mockHospitalRepository{exists: true}
	mockStaffRepo := &mockStaffRepository{exists: false}
	mockSessionRepo := &mockSessionRepository{}
	staffSvc := service.NewStaffService(mockHospRepo, mockStaffRepo, mockSessionRepo)

	input := service.CreateStaffInput{
		Username: "staff01",
		Password: "secretpassword",
		Hospital: "HOSP001",
	}

	output, err := staffSvc.CreateStaff(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if output.Staff == nil {
		t.Fatalf("expected output.Staff to not be nil")
	}
	if output.Staff.Staff.Username != input.Username || output.Staff.Staff.HospitalHN != input.Hospital {
		t.Errorf("output data does not match input: %+v", output.Staff)
	}
	if output.Staff.Staff.ID != "generated-staff-uuid" {
		t.Errorf("expected staff ID 'generated-staff-uuid', got %q", output.Staff.Staff.ID)
	}
	if output.Staff.Staff.HospitalName != "Test Hospital" {
		t.Errorf("expected hospital name 'Test Hospital', got %q", output.Staff.Staff.HospitalName)
	}
	if output.Staff.Session.ExpiresAt.IsZero() {
		t.Errorf("expected non-zero session expires_at")
	}
	if output.Token == "" {
		t.Errorf("expected non-empty JWT session token in output.Token")
	}

	// Verify that staff was saved to the repository
	if mockStaffRepo.createdStaff == nil {
		t.Fatalf("expected staff record to be saved into staff repository")
	}

	if mockStaffRepo.createdStaff.Username != "staff01" {
		t.Errorf("expected saved username 'staff01', got %q", mockStaffRepo.createdStaff.Username)
	}

	if mockStaffRepo.createdStaff.HospitalID != "00000000-0000-0000-0000-000000000001" {
		t.Errorf("expected saved hospital_id '00000000-0000-0000-0000-000000000001', got %q", mockStaffRepo.createdStaff.HospitalID)
	}

	// Verify password was hashed with bcrypt
	err = bcrypt.CompareHashAndPassword([]byte(mockStaffRepo.createdStaff.PasswordHash), []byte("secretpassword"))
	if err != nil {
		t.Errorf("expected password to be validly bcrypt hashed: %v", err)
	}

	// Verify that session was persisted into the session repository
	if mockSessionRepo.createdSession == nil {
		t.Fatalf("expected session record to be saved into session repository")
	}
	if mockSessionRepo.createdSession.StaffID != "generated-staff-uuid" {
		t.Errorf("expected session staff_id 'generated-staff-uuid', got %q", mockSessionRepo.createdSession.StaffID)
	}
	if mockSessionRepo.createdSession.Token != output.Token {
		t.Errorf("expected session token %q, got %q", output.Token, mockSessionRepo.createdSession.Token)
	}
}

func TestStaffService_CreateStaff_HospitalNotFound(t *testing.T) {
	mockHospRepo := &mockHospitalRepository{exists: false}
	mockStaffRepo := &mockStaffRepository{exists: false}
	staffSvc := service.NewStaffService(mockHospRepo, mockStaffRepo)

	input := service.CreateStaffInput{
		Username: "staff01",
		Password: "secretpassword",
		Hospital: "INVALID_HN",
	}

	_, err := staffSvc.CreateStaff(context.Background(), input)
	if !errors.Is(err, service.ErrHospitalNotFound) {
		t.Fatalf("expected ErrHospitalNotFound, got: %v", err)
	}

	if mockStaffRepo.createdStaff != nil {
		t.Errorf("expected no staff to be created when hospital not found")
	}
}

func TestStaffService_CreateStaff_StaffAlreadyExists(t *testing.T) {
	mockHospRepo := &mockHospitalRepository{exists: true}
	mockStaffRepo := &mockStaffRepository{exists: true}
	staffSvc := service.NewStaffService(mockHospRepo, mockStaffRepo)

	input := service.CreateStaffInput{
		Username: "duplicate_staff",
		Password: "secretpassword",
		Hospital: "HOSP001",
	}

	_, err := staffSvc.CreateStaff(context.Background(), input)
	if !errors.Is(err, service.ErrStaffAlreadyExists) {
		t.Fatalf("expected ErrStaffAlreadyExists, got: %v", err)
	}

	if mockStaffRepo.createdStaff != nil {
		t.Errorf("expected no staff to be created when staff already exists")
	}
}

func TestStaffService_CreateStaff_DBError(t *testing.T) {
	mockHospRepo := &mockHospitalRepository{err: errors.New("connection failed")}
	mockStaffRepo := &mockStaffRepository{exists: false}
	staffSvc := service.NewStaffService(mockHospRepo, mockStaffRepo)

	input := service.CreateStaffInput{
		Username: "staff01",
		Password: "secretpassword",
		Hospital: "HOSP001",
	}

	_, err := staffSvc.CreateStaff(context.Background(), input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestStaffService_CreateStaff_StaffCreateError(t *testing.T) {
	mockHospRepo := &mockHospitalRepository{exists: true}
	mockStaffRepo := &mockStaffRepository{createErr: errors.New("database insert error")}
	staffSvc := service.NewStaffService(mockHospRepo, mockStaffRepo)

	input := service.CreateStaffInput{
		Username: "staff01",
		Password: "secretpassword",
		Hospital: "HOSP001",
	}

	_, err := staffSvc.CreateStaff(context.Background(), input)
	if err == nil {
		t.Fatal("expected error when staff repo fails, got nil")
	}
}

func TestStaffService_CreateStaff_NilRepo(t *testing.T) {
	input := service.CreateStaffInput{
		Username: "staff01",
		Password: "secretpassword",
		Hospital: "HOSP001",
	}

	// Test nil hospital repo
	svc1 := service.NewStaffService(nil, &mockStaffRepository{})
	if _, err := svc1.CreateStaff(context.Background(), input); err == nil {
		t.Fatal("expected error when hospital repo is nil, got nil")
	}

	// Test nil staff repo
	svc2 := service.NewStaffService(&mockHospitalRepository{exists: true}, nil)
	if _, err := svc2.CreateStaff(context.Background(), input); err == nil {
		t.Fatal("expected error when staff repo is nil, got nil")
	}
}

func TestStaffService_LoginStaff_Success(t *testing.T) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("SecretPassword123!"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	existingStaff := &models.Staff{
		ID:           "test-staff-uuid",
		HospitalID:   "00000000-0000-0000-0000-000000000001",
		Username:     "somchai",
		PasswordHash: string(hashedPassword),
	}

	mockHospRepo := &mockHospitalRepository{exists: true}
	mockStaffRepo := &mockStaffRepository{findByHospitalAndUsernameStaff: existingStaff}
	mockSessionRepo := &mockSessionRepository{}
	staffSvc := service.NewStaffService(mockHospRepo, mockStaffRepo, mockSessionRepo)

	input := service.LoginStaffInput{
		Username: "somchai",
		Password: "SecretPassword123!",
		Hospital: "HOSP001",
	}

	output, err := staffSvc.LoginStaff(context.Background(), input)
	if err != nil {
		t.Fatalf("expected successful login, got: %v", err)
	}

	if output == nil || output.Staff == nil {
		t.Fatalf("expected non-nil output and staff response data")
	}
	if output.Staff.Staff.ID != existingStaff.ID {
		t.Errorf("expected staff ID %q, got %q", existingStaff.ID, output.Staff.Staff.ID)
	}
	if output.Staff.Staff.Username != input.Username {
		t.Errorf("expected username %q, got %q", input.Username, output.Staff.Staff.Username)
	}
	if output.Staff.Staff.HospitalHN != "HOSP001" {
		t.Errorf("expected hospital HN 'HOSP001', got %q", output.Staff.Staff.HospitalHN)
	}
	if output.Staff.Session.ExpiresAt.IsZero() {
		t.Errorf("expected non-zero expires_at")
	}
	if output.Token == "" {
		t.Errorf("expected non-empty token")
	}

	// Verify session persisted
	if mockSessionRepo.createdSession == nil {
		t.Fatalf("expected session to be created")
	}
	if mockSessionRepo.createdSession.StaffID != existingStaff.ID {
		t.Errorf("expected session staff_id %q, got %q", existingStaff.ID, mockSessionRepo.createdSession.StaffID)
	}
}

func TestStaffService_LoginStaff_UserNotFound(t *testing.T) {
	mockHospRepo := &mockHospitalRepository{exists: true}
	mockStaffRepo := &mockStaffRepository{findByHospitalAndUsernameStaff: nil}
	staffSvc := service.NewStaffService(mockHospRepo, mockStaffRepo)

	input := service.LoginStaffInput{
		Username: "unknown_user",
		Password: "SecretPassword123!",
		Hospital: "HOSP001",
	}

	_, err := staffSvc.LoginStaff(context.Background(), input)
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestStaffService_LoginStaff_WrongPassword(t *testing.T) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("CorrectPassword123!"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	existingStaff := &models.Staff{
		ID:           "test-staff-uuid",
		HospitalID:   "00000000-0000-0000-0000-000000000001",
		Username:     "somchai",
		PasswordHash: string(hashedPassword),
	}

	mockHospRepo := &mockHospitalRepository{exists: true}
	mockStaffRepo := &mockStaffRepository{findByHospitalAndUsernameStaff: existingStaff}
	staffSvc := service.NewStaffService(mockHospRepo, mockStaffRepo)

	input := service.LoginStaffInput{
		Username: "somchai",
		Password: "WrongPassword!",
		Hospital: "HOSP001",
	}

	_, err = staffSvc.LoginStaff(context.Background(), input)
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestStaffService_LoginStaff_HospitalNotFound(t *testing.T) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("SecretPassword123!"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	existingStaff := &models.Staff{
		ID:           "test-staff-uuid",
		HospitalID:   "non-existent-hosp",
		Username:     "somchai",
		PasswordHash: string(hashedPassword),
	}

	mockHospRepo := &mockHospitalRepository{exists: false}
	mockStaffRepo := &mockStaffRepository{findByHospitalAndUsernameStaff: existingStaff}
	staffSvc := service.NewStaffService(mockHospRepo, mockStaffRepo)

	input := service.LoginStaffInput{
		Username: "somchai",
		Password: "SecretPassword123!",
		Hospital: "UNKNOWN_HN",
	}

	_, err = staffSvc.LoginStaff(context.Background(), input)
	if !errors.Is(err, service.ErrHospitalNotFound) {
		t.Fatalf("expected ErrHospitalNotFound, got: %v", err)
	}
}

func TestStaffService_LoginStaff_DBError(t *testing.T) {
	mockHospRepo := &mockHospitalRepository{exists: true}
	mockStaffRepo := &mockStaffRepository{findByHospitalAndUsernameErr: errors.New("db disconnect")}
	staffSvc := service.NewStaffService(mockHospRepo, mockStaffRepo)

	input := service.LoginStaffInput{
		Username: "somchai",
		Password: "SecretPassword123!",
		Hospital: "HOSP001",
	}

	_, err := staffSvc.LoginStaff(context.Background(), input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestStaffService_LoginStaff_NilRepo(t *testing.T) {
	input := service.LoginStaffInput{
		Username: "somchai",
		Password: "SecretPassword123!",
		Hospital: "HOSP001",
	}

	svc1 := service.NewStaffService(nil, &mockStaffRepository{})
	if _, err := svc1.LoginStaff(context.Background(), input); err == nil {
		t.Fatal("expected error when hospital repo is nil, got nil")
	}

	svc2 := service.NewStaffService(&mockHospitalRepository{exists: true}, nil)
	if _, err := svc2.LoginStaff(context.Background(), input); err == nil {
		t.Fatal("expected error when staff repo is nil, got nil")
	}
}
