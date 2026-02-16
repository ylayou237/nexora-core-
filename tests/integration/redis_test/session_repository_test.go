package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/yvan/nexora-core/internal/adapters/secondary/redis"
	"github.com/yvan/nexora-core/internal/core/domain"
)

func TestRedisIntegration(t *testing.T) {
	// 1. Setup Miniredis
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	// 2. Initialisation de l'Adapter (doit être public !)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	adapter := &redis.Adapter{Client: client}
	repo := redis.NewSessionRepository(adapter)

	ctx := context.Background()

	// 3. Test de création de session
	t.Run("Full_Lifecycle", func(t *testing.T) {
		sID, _ := domain.NewSessionID("test-id")
		uID, _ := domain.NewUserID("user-id")
		mac, _ := domain.NewMAC("00:11:22:33:44:55")
		policy := domain.PolicySnapshot{DataQuota: 1000}

		session, _ := domain.NewActiveSession(sID, uID, "1.1.1.1", &mac, policy, time.Hour, &domain.FakeClock{})

		// Test Start
		if err := repo.StartSession(ctx, session); err != nil {
			t.Fatalf("StartSession failed: %v", err)
		}

		// Test Update via LUA
		delta := domain.UsageDelta{InputOctets: 500}
		if err := repo.UpdateUsage(ctx, sID, delta); err != nil {
			t.Errorf("UpdateUsage failed: %v", err)
		}
	})
}
