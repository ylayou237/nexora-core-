package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	nexoraPostgres "github.com/yvan/nexora-core/internal/adapters/secondary/postgres"
	"github.com/yvan/nexora-core/internal/core/domain"
)

// setupTestDB démarre un vrai conteneur Postgres Docker pour le test
func setupTestDB(t *testing.T) (*nexoraPostgres.Adapter, func()) {
	ctx := context.Background()

	// 1. Démarrage du Conteneur Postgres
	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("nexora_test"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	require.NoError(t, err)

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// 2. Connexion avec ton Adapter
	adapter, err := nexoraPostgres.NewAdapter(ctx, connStr)
	require.NoError(t, err)

	// 3. Application du Schéma (Migrations)
	schema := `
	CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

	CREATE TABLE tenants (
		id UUID PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		type VARCHAR(50) NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE users (
		id UUID PRIMARY KEY,
		tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
		username VARCHAR(64) NOT NULL,
		email VARCHAR(255),
		password_hash VARCHAR(255) NOT NULL,
		role VARCHAR(50) NOT NULL,
		mac_address VARCHAR(17),
		active BOOLEAN DEFAULT FALSE,
		expired_at TIMESTAMPTZ,
		max_sessions INTEGER DEFAULT 1,
		data_quota BIGINT DEFAULT 0,
		used_data BIGINT DEFAULT 0,
		version BIGINT DEFAULT 1,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW(),
		CONSTRAINT uq_users_tenant_username UNIQUE (tenant_id, username)
	);
	`
	_, err = adapter.Pool.Exec(ctx, schema)
	require.NoError(t, err)

	// Fonction de nettoyage
	cleanup := func() {
		adapter.Close()
		pgContainer.Terminate(ctx)
	}

	return adapter, cleanup
}

func TestUserRepositoryIntegration(t *testing.T) {
	// Préparation de l'environnement (Docker)
	adapter, cleanup := setupTestDB(t)
	defer cleanup()

	repo := nexoraPostgres.NewUserRepository(adapter)
	ctx := context.Background()
	clock := domain.NewRealClock()

	// Données de test
	tenantID1, _ := domain.NewTenantID("11111111-1111-1111-1111-111111111111")
	tenantID2, _ := domain.NewTenantID("22222222-2222-2222-2222-222222222222")

	// On crée les tenants en SQL brut pour satisfaire la clé étrangère
	_, err := adapter.Pool.Exec(ctx, "INSERT INTO tenants (id, name, type) VALUES ($1, 'Tenant A', 'provider'), ($2, 'Tenant B', 'reseller')", tenantID1.String(), tenantID2.String())
	require.NoError(t, err)

	t.Run("Create & GetByUsername : Succès", func(t *testing.T) {
		// Arrange
		id, _ := domain.NewUserID("aaaa1111-1234-5678-90ab-cdef12345678")
		username, _ := domain.NewUsername("alice")
		email, _ := domain.NewEmail("alice@nexora.com")
		hash, _ := domain.NewPasswordHash("hashed_secret")

		// ✅ CORRECTION : Utilisation de domain.NewUserParams
		user, err := domain.NewUser(domain.NewUserParams{
			ID:           id,
			Username:     username,
			Email:        email,
			PasswordHash: hash,
			Role:         domain.RoleProviderAdmin,
			TenantID:     tenantID1,
			MaxSessions:  5,
			DataQuota:    1000,
		}, clock)
		require.NoError(t, err)

		// Act
		err = repo.Create(ctx, user)
		assert.NoError(t, err)

		fetchedUser, err := repo.GetByUsername(ctx, tenantID1, username)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, user.ID(), fetchedUser.ID())
		assert.Equal(t, user.Email(), fetchedUser.Email())
		assert.Equal(t, uint64(1000), fetchedUser.DataQuota())
		assert.Equal(t, uint64(0), fetchedUser.UsedData())
	})

	t.Run("Isolation Multi-Tenant : Tenant A ne voit pas User de Tenant B", func(t *testing.T) {
		// Arrange
		id, _ := domain.NewUserID("bbbb2222-1234-5678-90ab-cdef12345678")
		username, _ := domain.NewUsername("bob")
		email, _ := domain.NewEmail("bob@nexora.com")
		hash, _ := domain.NewPasswordHash("secret")

		// ✅ CORRECTION : Utilisation de domain.NewUserParams
		bob, err := domain.NewUser(domain.NewUserParams{
			ID:           id,
			Username:     username,
			Email:        email,
			PasswordHash: hash,
			Role:         domain.RoleCustomer,
			TenantID:     tenantID2,
			MaxSessions:  1,
			DataQuota:    0,
		}, clock)
		require.NoError(t, err)
		require.NoError(t, repo.Create(ctx, bob))

		// Act : Cherche Bob dans Tenant 1 (ne doit pas exister)
		_, err = repo.GetByUsername(ctx, tenantID1, username)

		// Assert
		assert.ErrorIs(t, err, domain.ErrUserNotFound)

		// Act : Cherche Bob dans Tenant 2 (doit exister)
		found, err := repo.GetByUsername(ctx, tenantID2, username)
		assert.NoError(t, err)
		assert.Equal(t, bob.ID(), found.ID())
	})

	t.Run("Create : Conflit Username (Même Tenant)", func(t *testing.T) {
		// Arrange
		id, _ := domain.NewUserID("cccc3333-1234-5678-90ab-cdef12345678")
		username, _ := domain.NewUsername("alice") // Déjà pris par le premier test
		email, _ := domain.NewEmail("alice2@nexora.com")
		hash, _ := domain.NewPasswordHash("secret")

		// ✅ CORRECTION : Utilisation de domain.NewUserParams
		userDuplique, err := domain.NewUser(domain.NewUserParams{
			ID:           id,
			Username:     username,
			Email:        email,
			PasswordHash: hash,
			Role:         domain.RoleProviderAdmin,
			TenantID:     tenantID1,
			MaxSessions:  1,
			DataQuota:    0,
		}, clock)
		require.NoError(t, err)

		// Act
		err = repo.Create(ctx, userDuplique)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "conflict")
	})

	t.Run("Update : Optimistic Locking", func(t *testing.T) {
		// Arrange
		username, _ := domain.NewUsername("alice")

		// On recharge l'utilisateur existant depuis la DB
		user, err := repo.GetByUsername(ctx, tenantID1, username)
		require.NoError(t, err)

		// Simulation de concurrence : 2 versions du même objet
		userV1 := *user // Copie par valeur pour éviter les pointeurs partagés
		userV2 := *user

		// Act 1 : Modification V1 (doit réussir)
		err = userV1.Activate(clock)
		require.NoError(t, err)
		err = repo.Update(ctx, &userV1) // On passe l'adresse de la copie
		assert.NoError(t, err)

		// Act 2 : Modification V2 (doit échouer car version périmée)
		err = userV2.Expire(time.Now().Add(time.Hour), clock)
		require.NoError(t, err)
		err = repo.Update(ctx, &userV2) // On passe l'adresse de la copie

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "optimistic locking failure")
	})
}
