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

	// IMPORTANT : Assure-toi que le champ 'Client' dans ton struct Adapter est public (Majuscule)
	return &redis.Adapter{Client: client}, mr
}

func TestNasCache_Workflow(t *testing.T) {
	adapter, mr := setupAdapter(t)
	defer mr.Close()
	ctx := context.Background()

	// --- 1. Préparation des données ---
	ipStr := "192.168.88.1"
	ip := net.ParseIP(ipStr)

	nasID, _ := domain.NewNasID("nas-01")
	tenantID, _ := domain.NewTenantID("550e8400-e29b-41d4-a716-446655440000") // UUID valide

	// Création d'un NAS fictif
	nas, err := domain.NewNAS(
		nasID,
		tenantID,
		"identifier-01",
		"MikroTik-Router",
		ip,
		"secret123",
		&domain.FakeClock{},
	)
	if err != nil {
		t.Fatalf("Erreur création NAS: %v", err)
	}

	// --- 2. TEST : SetNas ---
	t.Run("SetNas", func(t *testing.T) {
		// On sauvegarde avec un TTL de 10 minutes
		err := adapter.SetNas(ctx, nas, 10*time.Minute)
		if err != nil {
			t.Fatalf("SetNas failed: %v", err)
		}

		// Vérification directe dans Miniredis que la clé existe
		// La clé doit correspondre à "nas:ip:" + IP
		expectedKey := "nas:ip:" + ipStr
		if !mr.Exists(expectedKey) {
			t.Errorf("La clé Redis %s n'a pas été créée", expectedKey)
		}
	})

	// --- 3. TEST : GetNas (Success) ---
	t.Run("GetNas_Success", func(t *testing.T) {
		// Note: Ton implémentation prend une 'string' pour l'IP, pas 'net.IP'
		result, err := adapter.GetNas(ctx, ipStr)
		if err != nil {
			t.Fatalf("GetNas failed: %v", err)
		}

		// Vérification des données
		if result.Identifier != "identifier-01" {
			t.Errorf("Identifier mismatch")
		}
		if result.Secret != "secret123" {
			t.Errorf("Secret mismatch")
		}
		// Vérification de l'IP (si ton objet NAS a une méthode ou un champ pour ça)
		// Attention : adapte selon que tu utilises .IP() ou .IPAddress
		if !result.IPAddress.Equal(ip) {
			t.Errorf("IP mismatch")
		}
	})

	// --- 4. TEST : GetNas (Not Found) ---
	t.Run("GetNas_NotFound", func(t *testing.T) {
		_, err := adapter.GetNas(ctx, "10.0.0.99") // IP inexistante

		// On vérifie que l'erreur retournée est bien celle définie dans ton package redis
		if err != redis.ErrNasNotFound {
			t.Errorf("Expected ErrNasNotFound, got: %v", err)
		}
	})

	// --- 5. TEST : TTL Expiration ---
	t.Run("TTL_Expiration", func(t *testing.T) {
		// On avance le temps de 11 minutes (TTL était 10min)
		mr.FastForward(11 * time.Minute)

		_, err := adapter.GetNas(ctx, ipStr)
		if err != redis.ErrNasNotFound {
			t.Error("Le NAS aurait dû expirer du cache")
		}
	})

	// --- 6. TEST : DeleteNas ---
	t.Run("DeleteNas", func(t *testing.T) {
		// On remet le NAS
		_ = adapter.SetNas(ctx, nas, time.Minute)

		// On le supprime
		err := adapter.DeleteNas(ctx, ipStr)
		if err != nil {
			t.Fatalf("DeleteNas failed: %v", err)
		}

		// On vérifie qu'il n'est plus là
		_, err = adapter.GetNas(ctx, ipStr)
		if err != redis.ErrNasNotFound {
			t.Error("Le NAS devrait être supprimé")
		}
	})
}
