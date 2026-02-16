package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/yvan/nexora-core/internal/core/domain"
	"github.com/yvan/nexora-core/internal/core/services"
)

// --- MOCK DU REPOSITORY ---
// Ce mock implémente l'intégralité de l'interface ports.CacheRepository
type MockCacheRepository struct {
	mock.Mock
}

// Implémentation de SetNX pour satisfaire l'interface
func (m *MockCacheRepository) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	args := m.Called(ctx, key, value, ttl)
	return args.Bool(0), args.Error(1)
}

func (m *MockCacheRepository) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockCacheRepository) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockCacheRepository) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

// --- TESTS DU SERVICE ---

func TestCacheService(t *testing.T) {
	mockRepo := new(MockCacheRepository)
	// On initialise le service en injectant notre mock
	service := services.NewCacheService(mockRepo)
	ctx := context.Background()

	t.Run("StoreSession doit utiliser le bon préfixe et les bons paramètres", func(t *testing.T) {
		jti := "uuid-session-123"
		userID := domain.NewUUID() // Si NewUUID() retourne un string ou un type UserID (string)
		ttl := 1 * time.Hour

		// CORRECTION : On retire .String() si userID est déjà un string ou un type basé sur string
		mockRepo.On("Set", ctx, "session:"+jti, string(userID), ttl).Return(nil).Once()

		err := service.StoreSession(ctx, jti, domain.UserID(userID), ttl)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
	t.Run("BlacklistToken doit marquer le token comme révoqué avec le bon TTL", func(t *testing.T) {
		jti := "token-to-kill"
		ttl := 24 * time.Hour

		// On s'aligne sur la réalité du code : on stocke "1"
		mockRepo.On("Set", ctx, "blacklist:"+jti, "1", ttl).Return(nil).Once()

		err := service.BlacklistToken(ctx, jti, ttl)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
	t.Run("IsTokenBlacklisted doit retourner true si le flag est présent", func(t *testing.T) {
		jti := "bad-token"

		mockRepo.On("Get", ctx, "blacklist:"+jti).Return("revoked", nil).Once()

		isBlacklisted, err := service.IsTokenBlacklisted(ctx, jti)

		assert.NoError(t, err)
		assert.True(t, isBlacklisted)
	})

	t.Run("IsTokenBlacklisted doit retourner false si Redis renvoie l'erreur 'nil'", func(t *testing.T) {
		jti := "clean-token"

		// Simulation de l'erreur redis.Nil via une erreur textuelle
		mockRepo.On("Get", ctx, "blacklist:"+jti).Return("", errors.New("redis: nil")).Once()

		isBlacklisted, err := service.IsTokenBlacklisted(ctx, jti)

		assert.NoError(t, err)
		assert.False(t, isBlacklisted)
	})
}
