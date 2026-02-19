package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/yvan/nexora-core/internal/core/domain"
	"github.com/yvan/nexora-core/internal/core/services"
	"golang.org/x/crypto/bcrypt"
)

// --- 1. CONFIGURATION & HELPERS ---

var (
	testTenantID, _ = domain.NewTenantID("550e8400-e29b-41d4-a716-446655440000")
	testUserID, _   = domain.NewUserID("550e8400-e29b-41d4-a716-446655440001")
	testUsername, _ = domain.NewUsername("yvan")
	testJWTSecret   = "test-secret-key-very-long-32-chars-!!"
	// ✅ On définit les TTL pour les tests
	testAccessTTL  = 15 * time.Minute
	testRefreshTTL = 7 * 24 * time.Hour
)

func setupTestUser(clock domain.Clock, active bool, expired bool, usedData uint64) *domain.User {
	password := "password123"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	passHash, _ := domain.NewPasswordHash(string(hash))

	var expiry *time.Time
	if expired {
		past := clock.Now().Add(-1 * time.Hour)
		expiry = &past
	}

	u, _ := domain.RehydrateUser(domain.UserSnapshot{
		ID:           testUserID,
		Username:     testUsername,
		Email:        "yvan@test.com",
		PasswordHash: passHash,
		Role:         domain.RoleCustomer,
		TenantID:     testTenantID,
		Active:       active,
		ExpiredAt:    expiry,
		MaxSessions:  1,
		DataQuota:    1000,
		UsedData:     usedData,
		Version:      1,
		CreatedAt:    clock.Now(),
		UpdatedAt:    clock.Now(),
	})
	return u
}

// --- 2. TESTS UNITAIRES ---

func TestLoginSuccess(t *testing.T) {
	clock := domain.NewFakeClock(time.Now())
	userRepo := &MockUserRepo{User: setupTestUser(clock, true, false, 0)}

	// ✅ AJOUT DES TTL ICI
	authService := services.NewAuthService(userRepo, &MockSessionRepo{}, &MockProtectionRepo{}, clock, testJWTSecret, testAccessTTL, testRefreshTTL)

	tokens, err := authService.Login(context.Background(), testTenantID, testUsername, "password123")

	if err != nil {
		t.Errorf("Le login aurait dû réussir, erreur: %v", err)
	}
	if tokens.AccessToken == "" {
		t.Error("L'access token ne doit pas être vide")
	}
}

func TestLoginInvalidPassword(t *testing.T) {
	clock := domain.NewFakeClock(time.Now())
	userRepo := &MockUserRepo{User: setupTestUser(clock, true, false, 0)}
	authService := services.NewAuthService(userRepo, &MockSessionRepo{}, &MockProtectionRepo{}, clock, testJWTSecret, testAccessTTL, testRefreshTTL)

	_, err := authService.Login(context.Background(), testTenantID, testUsername, "mauvais-pass")

	if err != domain.ErrInvalidCredentials {
		t.Errorf("Attendu: ErrInvalidCredentials, obtenu: %v", err)
	}
}

func TestLoginInactiveAccount(t *testing.T) {
	clock := domain.NewFakeClock(time.Now())
	userRepo := &MockUserRepo{User: setupTestUser(clock, false, false, 0)}
	authService := services.NewAuthService(userRepo, &MockSessionRepo{}, &MockProtectionRepo{}, clock, testJWTSecret, testAccessTTL, testRefreshTTL)

	_, err := authService.Login(context.Background(), testTenantID, testUsername, "password123")

	if err != domain.ErrUserInactive {
		t.Errorf("Attendu: ErrUserInactive, obtenu: %v", err)
	}
}

func TestLoginExpiredAccount(t *testing.T) {
	clock := domain.NewFakeClock(time.Now())
	userRepo := &MockUserRepo{User: setupTestUser(clock, true, true, 0)}
	authService := services.NewAuthService(userRepo, &MockSessionRepo{}, &MockProtectionRepo{}, clock, testJWTSecret, testAccessTTL, testRefreshTTL)

	_, err := authService.Login(context.Background(), testTenantID, testUsername, "password123")

	if err != domain.ErrUserExpired {
		t.Errorf("Attendu: ErrUserExpired, obtenu: %v", err)
	}
}

func TestLoginAccountLockedByBruteForce(t *testing.T) {
	clock := domain.NewFakeClock(time.Now())
	userRepo := &MockUserRepo{User: setupTestUser(clock, true, false, 0)}
	protRepo := &MockProtectionRepo{
		IsLockedFunc: func() (bool, time.Duration) { return true, 15 * time.Minute },
	}
	authService := services.NewAuthService(userRepo, &MockSessionRepo{}, protRepo, clock, testJWTSecret, testAccessTTL, testRefreshTTL)

	_, err := authService.Login(context.Background(), testTenantID, testUsername, "password123")

	if err != domain.ErrAccountLocked {
		t.Errorf("Attendu: ErrAccountLocked, obtenu: %v", err)
	}
}

func TestLoginUserNotFound(t *testing.T) {
	clock := domain.NewFakeClock(time.Now())
	userRepo := &MockUserRepo{Err: domain.ErrUserNotFound}
	authService := services.NewAuthService(userRepo, &MockSessionRepo{}, &MockProtectionRepo{}, clock, testJWTSecret, testAccessTTL, testRefreshTTL)

	_, err := authService.Login(context.Background(), testTenantID, testUsername, "password123")

	if err != domain.ErrInvalidCredentials {
		t.Errorf("UserNotFound doit renvoyer ErrInvalidCredentials. Obtenu: %v", err)
	}
}

// --- 3. MOCKS --- (Inchangés)

type MockUserRepo struct {
	User *domain.User
	Err  error
}

func (m *MockUserRepo) GetByUsername(ctx context.Context, tID domain.TenantID, u domain.Username) (*domain.User, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.User, nil
}
func (m *MockUserRepo) GetByID(ctx context.Context, id domain.UserID) (*domain.User, error) {
	return m.User, m.Err
}
func (m *MockUserRepo) Create(ctx context.Context, u *domain.User) error   { return nil }
func (m *MockUserRepo) Update(ctx context.Context, u *domain.User) error   { return nil }
func (m *MockUserRepo) Delete(ctx context.Context, id domain.UserID) error { return nil }

type MockSessionRepo struct{}

func (m *MockSessionRepo) StartSession(ctx context.Context, s *domain.ActiveSession) error {
	return nil
}
func (m *MockSessionRepo) TerminateSession(ctx context.Context, id domain.SessionID) error {
	return nil
}
func (m *MockSessionRepo) GetByID(ctx context.Context, id domain.SessionID) (*domain.ActiveSession, error) {
	return nil, nil
}
func (m *MockSessionRepo) UpdateUsage(ctx context.Context, id domain.SessionID, delta domain.UsageDelta) error {
	return nil
}
func (m *MockSessionRepo) Exists(ctx context.Context, id domain.SessionID) (bool, error) {
	return true, nil
}

type MockProtectionRepo struct {
	IsLockedFunc func() (bool, time.Duration)
	Attempts     int64
}

func (m *MockProtectionRepo) IsLocked(ctx context.Context, key string) (bool, time.Duration, error) {
	if m.IsLockedFunc != nil {
		l, d := m.IsLockedFunc()
		return l, d, nil
	}
	return false, 0, nil
}
func (m *MockProtectionRepo) RecordFailedAttempt(ctx context.Context, key string) (int64, error) {
	m.Attempts++
	return m.Attempts, nil
}
func (m *MockProtectionRepo) ClearAttempts(ctx context.Context, key string) error {
	m.Attempts = 0
	return nil
}
