package domain_test

import (
	"testing"
	"time"

	"github.com/yvan/nexora-core/internal/core/domain"
)

func TestUserCreationActivation(t *testing.T) {
	// 1. DÉFINITION D'UN TEMPS FIXE
	// On crée une date précise. Comme on utilise un FakeClock, le test
	// ne dépendra jamais de l'heure réelle de ton ordinateur.
	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := &FakeClock{now: fixedTime} // Utilisation du champ 'now' défini dans setup_test.go

	// 2. INITIALISATION DES VALUE OBJECTS
	// On s'assure que les données de base respectent les règles de format (UUID, Email, Hash).
	tenantID, _ := domain.NewTenantID("123e4567-e89b-12d3-a456-426614174000")
	userID, _ := domain.NewUserID("123e4567-e89b-12d3-a456-426614174001")
	username, _ := domain.NewUsername("yvan")
	email, _ := domain.NewEmail("yvan@example.com")
	passwordHash, _ := domain.NewPasswordHash("12345678901234567890123456789012")

	// On simule un Tenant existant (l'entreprise parente)
	tenant := domain.RehydrateTenant(tenantID, nil, "OperatorTenant", domain.TenantOperator, false)

	// 3. TEST DE LA CRÉATION (FACTORY)
	// ✅ Correction : Utilisation du Parameter Object pour satisfaire le compilateur
	user, err := domain.NewUser(domain.NewUserParams{
		ID:           userID,
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         domain.RoleSuperAdmin,
		TenantID:     tenant.ID(),
		MaxSessions:  5,
		DataQuota:    1024 * 1024,
	}, clock)

	if err != nil {
		t.Fatalf("Le domaine a refusé de créer l'utilisateur : %v", err)
	}

	// VÉRIFICATION DE LA SÉCURITÉ PAR DÉFAUT
	// Un utilisateur ne doit JAMAIS être actif dès sa création (anti-spam/validation).
	if user.IsActive() {
		t.Errorf("ERREUR : L'utilisateur est actif par défaut, c'est une faille de sécurité")
	}

	// 4. TEST DU COMPORTEMENT MÉTIER (ACTIVATION)
	// On simule l'action d'activer un compte (ex: après validation email).
	if err := user.Activate(clock); err != nil {
		t.Fatalf("L'activation a échoué : %v", err)
	}

	// 5. ASSERTION FINALE
	// On vérifie que l'état interne a bien basculé à "Actif".
	if !user.IsActive() {
		t.Errorf("ERREUR : L'utilisateur devrait être actif après l'appel à Activate()")
	}
}
