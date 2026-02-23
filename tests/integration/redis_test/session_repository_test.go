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

// setupTestSession centralise la création de données pour réduire la complexité du test principal
func setupTestSession(now time.Time) (*domain.ActiveSession, domain.SessionID) {
	sID := domain.SessionID(domain.NewUUID())
	uID := domain.UserID(domain.NewUUID())
	mac, _ := domain.NewMAC("00:11:22:33:44:55")
	policy := domain.PolicySnapshot{DataQuota: 1000}

	// ✅ Utilise le constructeur : c'est propre et validé par ton package domain
	clock := domain.NewFakeClock(now)

	session, _ := domain.NewActiveSession(sID, uID, "1.1.1.1", &mac, policy, time.Hour, clock)

	return session, sID
}
func TestSessionRepositoryIntegration(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	repo := redis.NewSessionRepository(client)
	ctx := context.Background()
	session, sID := setupTestSession(time.Now())

	t.Run("Lifecycle", func(t *testing.T) {
		// 1. Persistance
		if err := repo.StartSession(ctx, session); err != nil {
			t.Errorf("StartSession failed: %v", err)
		}

		// 2. Mise à jour via LUA script
		delta := domain.UsageDelta{InputOctets: 100}
		if err := repo.UpdateUsage(ctx, sID, delta); err != nil {
			t.Errorf("UpdateUsage failed: %v", err)
		}

		// 3. Récupération et vérification
		got, err := repo.GetByID(ctx, sID)
		if err != nil || got.InputOctets != 100 {
			t.Errorf("Data mismatch after update: got %d", got.InputOctets)
		}

		// 4. Suppression
		_ = repo.TerminateSession(ctx, sID)
		if _, err := repo.GetByID(ctx, sID); err == nil {
			t.Error("Session should be deleted")
		}
	})

	t.Run("Infrastructure", func(t *testing.T) {
		if err := repo.Health(ctx); err != nil {
			t.Errorf("Health check failed: %v", err)
		}
	})
}
