package domain_test

import (
	"testing"
	"time"

	"github.com/yvan/nexora-core/internal/core/domain"
)

func TestUserIdentityLifecycle(t *testing.T) {
	clock := &FakeClock{currentTime: time.Now()}

	// 1. Setup Value Objects
	uid, _ := domain.NewUserID("550e8400-e29b-41d4-a716-446655440000")
	email, _ := domain.NewEmail("alice@example.com")
	username, _ := domain.NewUsername("alice")
	passHash, _ := domain.NewPasswordHash("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy")

	// Setup Hiérarchie Tenant
	opID, _ := domain.NewTenantID("11111111-1111-4111-8111-111111111111")
	op, _ := domain.NewTenant(opID, "Op", domain.TenantOperator, nil, false)

	provID, _ := domain.NewTenantID("22222222-2222-4222-8222-222222222222")
	_, _ = domain.NewTenant(provID, "Prov", domain.TenantProvider, op, true)

	// 2. Création de l'Agrégat User avec NewUserParams
	// ✅ CORRECTION : Le compilateur veut une struct NewUserParams
	user, err := domain.NewUser(domain.NewUserParams{
		ID:           uid,
		Username:     username,
		Email:        email,
		PasswordHash: passHash,
		Role:         domain.RoleProviderAdmin,
		TenantID:     provID,
		MaxSessions:  5,
		DataQuota:    1024 * 1024 * 10,
	}, clock)

	if err != nil {
		t.Fatalf("Creation failed: %v", err)
	}

	// 3. Test de l'Invariance Initiale
	if user.IsActive() {
		t.Error("Security Breach: User should be inactive by default")
	}
	if user.Version() != 1 {
		t.Errorf("Initial version should be 1, got %d", user.Version())
	}

	// 4. Test de Transition d'État : Activation
	if err := user.Activate(clock); err != nil {
		t.Errorf("Activation failed: %v", err)
	}
	if !user.IsActive() {
		t.Error("User state should be active")
	}

	if user.Version() != 2 {
		t.Errorf("Version should be 2 after activation, got %d", user.Version())
	}

	// 5. Test de Transition d'État : Expiration temporelle
	future := clock.Now().Add(24 * time.Hour)
	user.Expire(future, clock)

	// Time Travel
	clock.Advance(25 * time.Hour)

	if !user.IsExpired(clock) {
		t.Error("Business Logic Failure: User should be detected as expired")
	}

	// 6. Test du Guard (CanAuthenticate)
	if err := user.CanAuthenticate(clock); err != domain.ErrUserExpired {
		t.Errorf("Security Guard Failure: Expected ErrUserExpired, got %v", err)
	}
}
