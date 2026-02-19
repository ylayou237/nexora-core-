package redis_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/yvan/nexora-core/internal/adapters/secondary/redis"
	"github.com/yvan/nexora-core/internal/core/domain"
)

// setupAdapter prépare l'environnement de test avec Miniredis
func setupAdapter(t *testing.T) (*redis.Adapter, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	return &redis.Adapter{Client: client}, mr
}

// createTestNAS est un helper pour le domaine
func createTestNAS(t *testing.T, ip net.IP) *domain.NAS {
	nasID, _ := domain.NewNasID("nas-01")
	tenantID, _ := domain.NewTenantID("550e8400-e29b-41d4-a716-446655440000")

	nas, err := domain.NewNAS(domain.NewNASParams{
		ID:         nasID,
		TenantID:   tenantID,
		Identifier: "identifier-01",
		ShortName:  "MikroTik-Router",
		IP:         ip,
		Secret:     "secret123",
	}, &domain.FakeClock{})

	if err != nil {
		t.Fatalf("Erreur création NAS: %v", err)
	}
	return nas
}

// --- TESTS CORRIGÉS (NOMMAGE CAMELCASE) ---

func TestNasCachePersistence(t *testing.T) {
	adapter, mr := setupAdapter(t)
	defer mr.Close()
	ctx := context.Background()

	ipStr := "192.168.88.1"
	nas := createTestNAS(t, net.ParseIP(ipStr))

	t.Run("Sauvegarde et récupération", func(t *testing.T) {
		_ = adapter.SetNas(ctx, nas, 10*time.Minute)

		result, err := adapter.GetNas(ctx, ipStr)
		if err != nil {
			t.Fatalf("Le NAS aurait dû être trouvé: %v", err)
		}
		if result.Secret != "secret123" {
			t.Error("Données corrompues lors de la récupération")
		}
	})
}

func TestNasCacheExpiration(t *testing.T) {
	adapter, mr := setupAdapter(t)
	defer mr.Close()
	ctx := context.Background()

	ipStr := "10.0.0.1"
	nas := createTestNAS(t, net.ParseIP(ipStr))

	t.Run("Le cache doit expirer", func(t *testing.T) {
		_ = adapter.SetNas(ctx, nas, 1*time.Minute)

		// On simule le passage du temps dans Redis
		mr.FastForward(2 * time.Minute)

		_, err := adapter.GetNas(ctx, ipStr)
		if err != redis.ErrNasNotFound {
			t.Errorf("Attendu: ErrNasNotFound, Obtenu: %v", err)
		}
	})
}

func TestNasCacheDeletion(t *testing.T) {
	adapter, mr := setupAdapter(t)
	defer mr.Close()
	ctx := context.Background()

	ipStr := "172.16.0.1"
	nas := createTestNAS(t, net.ParseIP(ipStr))

	t.Run("Suppression manuelle", func(t *testing.T) {
		_ = adapter.SetNas(ctx, nas, 10*time.Minute)

		if err := adapter.DeleteNas(ctx, ipStr); err != nil {
			t.Fatalf("Échec suppression: %v", err)
		}

		if mr.Exists("nas:ip:" + ipStr) {
			t.Error("La clé existe encore dans Redis après suppression")
		}
	})
}
