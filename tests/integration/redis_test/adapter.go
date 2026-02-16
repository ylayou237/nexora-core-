package redis_test

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/yvan/nexora-core/internal/adapters/secondary/redis"
)

func TestAdapter_Connection(t *testing.T) {
	// 1. Démarrage du serveur Redis simulé (Miniredis)
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Impossible de lancer miniredis: %v", err)
	}
	defer mr.Close()

	// 2. Initialisation du client Redis
	client := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})

	// 3. Création de l'Adapter Nexora
	// IMPORTANT : Assure-toi que le champ 'Client' est Public (Majuscule) dans adapter.go
	adapter := &redis.Adapter{Client: client}

	// 4. Test du Ping (Vérifie que la connexion est active)
	ctx := context.Background()
	if err := adapter.Client.Ping(ctx).Err(); err != nil {
		t.Errorf("Le Ping vers Redis a échoué: %v", err)
	}
}
