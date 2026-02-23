package services_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
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
)

// setupAuthService initialise le service avec des mocks robustes
func setupAuthService(deps services.AuthDependencies) *services.AuthService {
	if deps.Clock == nil {
		deps.Clock = domain.NewFakeClock(time.Now())
	}
	if deps.UserRepo == nil {
		deps.UserRepo = &MockUserRepo{}
	}
	if deps.JwksService == nil {
		deps.JwksService = NewMockJWKSService()
	}
	if deps.SessionRepo == nil {
		deps.SessionRepo = &MockSessionRepo{}
	}
	if deps.ProtectionRepo == nil {
		deps.ProtectionRepo = &MockProtectionRepo{}
	}
	if deps.AuditRepo == nil {
		deps.AuditRepo = &MockAuditRepo{}
	}
	if deps.RefreshRepo == nil {
		deps.RefreshRepo = &MockRefreshRepo{}
	}
	if deps.RefreshHasher == nil {
		deps.RefreshHasher = &MockHasher{}
	}
	if deps.RiskEngine == nil {
		deps.RiskEngine = &MockRiskEngine{}
	}
	if deps.OtpService == nil {
		deps.OtpService = &MockOTPService{}
	}

	deps.Config = services.DefaultAuthConfig()
	deps.Config.Issuer = "nexora-test"

	return services.NewAuthService(deps)
}

func setupTestUser(clock domain.Clock, active bool) *domain.User {
	password := "password123"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	passHash, _ := domain.NewPasswordHash(string(hash))

	// Utilisation de la clock pour définir une expiration valide
	expiry := clock.Now().Add(8760 * time.Hour)

	u, _ := domain.RehydrateUser(domain.UserSnapshot{
		ID:           testUserID,
		Username:     testUsername,
		PasswordHash: passHash,
		TenantID:     testTenantID,
		Active:       active,
		ExpiredAt:    &expiry,
		DataQuota:    1000,
	})
	return u
}

// --- 2. TESTS UNITAIRES ---

func TestLoginSuccess(t *testing.T) {
	clock := domain.NewFakeClock(time.Now())
	user := setupTestUser(clock, true)

	deps := services.AuthDependencies{
		UserRepo: &MockUserRepo{User: user},
		Clock:    clock,
	}
	authService := setupAuthService(deps)

	req := services.LoginRequest{
		TenantID: testTenantID,
		Username: testUsername,
		Password: "password123",
		IPAddr:   "127.0.0.1",
		TraceID:  "test-trace",
	}

	tokens, mfa, err := authService.Login(context.Background(), req)

	if err != nil {
		t.Fatalf("Le login aurait dû réussir, erreur: %v", err)
	}
	if mfa {
		t.Error("MFA ne devrait pas être requis ici")
	}
	if tokens == nil || tokens.AccessToken == "" {
		t.Error("AccessToken manquant")
	}
}

// --- 3. MOCKS (CONFORMITÉ GOPLS & SONARLINT) ---

type MockJWKSService struct {
	key *rsa.PrivateKey
}

func NewMockJWKSService() *MockJWKSService {
	k, _ := rsa.GenerateKey(rand.Reader, 2048)
	return &MockJWKSService{key: k}
}

func (m *MockJWKSService) GetCurrentPrivateKey() (*rsa.PrivateKey, string) {
	return m.key, "test-kid"
}
func (m *MockJWKSService) GetPublicKeyByKid(_ string) (*rsa.PublicKey, error) {
	return &m.key.PublicKey, nil
}
func (m *MockJWKSService) GetJWKS() services.JWKS { return services.JWKS{} }
func (m *MockJWKSService) Stop() {
	// No-op: rien à libérer pour le mock
}

type MockUserRepo struct {
	User *domain.User
	Err  error
}

func (m *MockUserRepo) GetByUsername(_ context.Context, _ domain.TenantID, _ domain.Username) (*domain.User, error) {
	return m.User, m.Err
}
func (m *MockUserRepo) GetByID(_ context.Context, _ domain.UserID) (*domain.User, error) {
	return m.User, m.Err
}
func (m *MockUserRepo) Create(_ context.Context, _ *domain.User) error  { return nil }
func (m *MockUserRepo) Update(_ context.Context, _ *domain.User) error  { return nil }
func (m *MockUserRepo) Delete(_ context.Context, _ domain.UserID) error { return nil }
func (m *MockUserRepo) Health(_ context.Context) error                  { return nil }

type MockSessionRepo struct{}

func (m *MockSessionRepo) StartSession(_ context.Context, _ *domain.ActiveSession) error { return nil }
func (m *MockSessionRepo) TerminateSession(_ context.Context, _ domain.SessionID) error  { return nil }
func (m *MockSessionRepo) GetByID(_ context.Context, _ domain.SessionID) (*domain.ActiveSession, error) {
	return nil, nil
}
func (m *MockSessionRepo) UpdateUsage(_ context.Context, _ domain.SessionID, _ domain.UsageDelta) error {
	return nil
}
func (m *MockSessionRepo) Exists(_ context.Context, _ domain.SessionID) (bool, error) {
	return true, nil
}
func (m *MockSessionRepo) Health(_ context.Context) error { return nil }

type MockProtectionRepo struct{}

func (m *MockProtectionRepo) RecordFailedAttempt(_ context.Context, _ string) (int64, error) {
	return 0, nil
}
func (m *MockProtectionRepo) ClearAttempts(_ context.Context, _ string) error { return nil }
func (m *MockProtectionRepo) IsLocked(_ context.Context, _ string) (bool, time.Duration, error) {
	return false, 0, nil
}

type MockAuditRepo struct{}

func (m *MockAuditRepo) LogEvent(_ context.Context, _ *domain.AuditLog) error { return nil }
func (m *MockAuditRepo) FindLogs(_ context.Context, _ domain.AuditFilter) ([]domain.AuditLog, error) {
	return nil, nil
}
func (m *MockAuditRepo) Health(_ context.Context) error { return nil }

type MockRefreshRepo struct{}

func (m *MockRefreshRepo) Save(_ context.Context, _ *domain.RefreshToken) error { return nil }
func (m *MockRefreshRepo) GetByHash(_ context.Context, _ string) (*domain.RefreshToken, error) {
	return nil, nil
}
func (m *MockRefreshRepo) Rotate(_ context.Context, _ domain.RefreshTokenID, _ time.Time, _, _, _ string) (*domain.RefreshToken, error) {
	return &domain.RefreshToken{TokenRaw: "new_refresh", UserID: testUserID}, nil
}
func (m *MockRefreshRepo) Revoke(_ context.Context, _ domain.RefreshTokenID) error { return nil }
func (m *MockRefreshRepo) RevokeFamily(_ context.Context, _ domain.TokenFamilyID, _ time.Time, _ string) error {
	return nil
}
func (m *MockRefreshRepo) Health(_ context.Context) error { return nil }

type MockHasher struct{}

func (m *MockHasher) Hash(_ string) string { return "hashed" }

type MockRiskEngine struct{}

func (m *MockRiskEngine) ComputeRisk(_ context.Context, _ *domain.User, _, _, _ string) (float64, error) {
	return 0.1, nil
}

type MockOTPService struct{}

func (m *MockOTPService) SendOTP(_ context.Context, _ *domain.User, _ string) error  { return nil }
func (m *MockOTPService) VerifyOTP(_ context.Context, _ *domain.User, _ string) bool { return true }
