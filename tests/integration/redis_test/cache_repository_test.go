package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	nexoraRedis "github.com/yvan/nexora-core/internal/adapters/secondary/redis"
)

func TestCacheRepositoryIntegration(t *testing.T) {
	// 1. SETUP : MiniRedis
	mr := miniredis.RunT(t)
	ctx := context.Background()

	// 2. SETUP : Ton Adapter (La connexion)
	adapter, err := nexoraRedis.NewAdapter(
		ctx,
		"",
		[]string{mr.Addr()},
		"",
	)
	require.NoError(t, err)
	defer adapter.Close()

	// 3. SETUP : Ton Repository (L'objet à tester)
	// ✅ CORRECTION : C'est cet objet qu'on teste maintenant, pas l'adapter directement
	repo := nexoraRedis.NewCacheRepository(adapter)

	// --- TESTS ---

	t.Run("Action Set et Get : Succès", func(t *testing.T) {
		key := "session:test:1"
		value := "user_99"
		ttl := 5 * time.Minute

		// ✅ On appelle repo.Set, plus adapter.Set
		err := repo.Set(ctx, key, value, ttl)
		assert.NoError(t, err)

		got, err := repo.Get(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, value, got)
	})

	t.Run("Action Get : Clé inexistante", func(t *testing.T) {
		// ✅ Utilisation du repo
		_, err := repo.Get(ctx, "non_existent_key")
		assert.ErrorIs(t, err, redis.Nil)
	})

	t.Run("Action Expiration : TTL respecté", func(t *testing.T) {
		key := "expiring_soon"
		ttl := 1 * time.Second

		repo.Set(ctx, key, "temporary", ttl)

		// On avance le temps dans miniredis
		mr.FastForward(2 * time.Second)

		_, err = repo.Get(ctx, key)
		assert.ErrorIs(t, err, redis.Nil)
	})

	t.Run("Action Delete : Succès", func(t *testing.T) {
		key := "to_delete"
		repo.Set(ctx, key, "val", time.Hour)

		err := repo.Delete(ctx, key)
		assert.NoError(t, err)

		_, err = repo.Get(ctx, key)
		assert.ErrorIs(t, err, redis.Nil)
	})
}
