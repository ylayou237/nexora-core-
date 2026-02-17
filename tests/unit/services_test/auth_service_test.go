package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/yvan/nexora-core/internal/core/domain"
	"github.com/yvan/nexora-core/internal/core/services"
	"golang.org/x/crypto/bcrypt"
)

// --- MOCKS (Bouchons) ---
// En architecture hexagonale, on ne teste pas la base de données réelle dans les tests unitaires.
// On crée des "Mocks" qui simulent le comportement des repositories.

// MockUserRepo simule ports.UserRepository pour manipuler les données utilisateur sans Postgres.
type MockUserRepo struct {
	User *domain.User // L'utilisateur que le mock va retourner
	Err  error        // L'erreur éventuelle (ex: utilisateur non trouvé)
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

// Ces méthodes sont nécessaires pour satisfaire l'interface UserRepository même si inutilisées ici.
func (m *MockUserRepo) Create(ctx context.Context, u *domain.User) error   { return nil }
func (m *MockUserRepo) Update(ctx context.Context, u *domain.User) error   { return nil }
func (m *MockUserRepo) Delete(ctx context.Context, id domain.UserID) error { return nil }

// MockSessionRepo simule ports.SessionRepository pour tester la gestion des sessions sans Redis.
type MockSessionRepo struct {
	StartErr error
}

func (m *MockSessionRepo) StartSession(ctx context.Context, s *domain.ActiveSession) error {
	return m.StartErr
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

// --- TESTS DE LOGIQUE D'AUTHENTIFICATION ---

func TestAuthService_Login(t *testing.T) {
	// 1. INITIALISATION DU CONTEXTE DE TEST (ARANGE)
	ctx := context.Background()
	clock := &domain.FakeClock{} // Utilisation du FakeClock pour contrôler le temps
	jwtSecret := "super-secret-key"

	// Création des Value Objects nécessaires
	tenantID, _ := domain.NewTenantID("550e8400-e29b-41d4-a716-446655440000")
	userID, _ := domain.NewUserID("550e8400-e29b-41d4-a716-446655440001")
	email, _ := domain.NewEmail("yvan@example.com")
	username, _ := domain.NewUsername("yvan")

	// Simulation d'un mot de passe haché (Bcrypt) tel qu'il serait stocké en DB
	password := "password123"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	passHash, _ := domain.NewPasswordHash(string(hash))

	now := time.Now()

	// 2. RECONSTRUCTION D'UN UTILISATEUR VALIDE (Entité de domaine)
	// On utilise RehydrateUser car l'utilisateur est censé "exister déjà" en base.
	// 2. RECONSTRUCTION D'UN UTILISATEUR VALIDE
	validUser, err := domain.RehydrateUser(
		userID,              // 1. UserID
		username,            // 2. Username
		email,               // 3. Email
		passHash,            // 4. PasswordHash
		nil,                 // 5. *MAC (Hardware lock)
		domain.RoleCustomer, // 6. Role
		tenantID,            // 7. TenantID
		true,                // 8. IsActive
		nil,                 // 9. *ExpirationDate
		0,                   // 10. SessionLimit (int)
		0,                   // 11. DataQuota (uint64)
		0,                   // 12. UsedQuota (uint64) - THIS WAS MISSING
		1,                   // 13. Version (uint64)
		now,                 // 14. CreatedAt (time.Time)
		now,                 // 15. UpdatedAt (time.Time)
	)
	if err != nil {
		t.Fatalf("Erreur critique: impossible de créer l'utilisateur de test: %v", err)
	}

	// --- SCÉNARIO 1 : CONNEXION RÉUSSIE ---
	t.Run("Success_Login", func(t *testing.T) {
		// On configure les mocks pour simuler un comportement idéal
		userRepo := &MockUserRepo{User: validUser}
		sessionRepo := &MockSessionRepo{}
		authService := services.NewAuthService(userRepo, sessionRepo, clock, jwtSecret)

		// ACTION : Tentative de login
		tokens, err := authService.Login(ctx, tenantID, "yvan", password)

		// ASSERTION : On vérifie qu'il n'y a pas d'erreur et que les tokens sont générés
		if err != nil {
			t.Fatalf("Succès attendu, mais erreur obtenue: %v", err)
		}
		if tokens.AccessToken == "" || tokens.RefreshToken == "" {
			t.Error("Les jetons Access ou Refresh ne doivent pas être vides")
		}
	})

	// --- SCÉNARIO 2 : ÉCHEC PAR MAUVAIS MOT DE PASSE ---
	t.Run("Invalid_Password", func(t *testing.T) {
		userRepo := &MockUserRepo{User: validUser}
		sessionRepo := &MockSessionRepo{}
		authService := services.NewAuthService(userRepo, sessionRepo, clock, jwtSecret)

		// ACTION : Tentative avec un mauvais mot de passe
		_, err := authService.Login(ctx, tenantID, "yvan", "wrong-password")

		// ASSERTION : On vérifie que le service retourne bien l'erreur métier de sécurité
		if err != domain.ErrInvalidCredentials {
			t.Errorf("Attendu: ErrInvalidCredentials, Obtenu: %v", err)
		}
	})
}
