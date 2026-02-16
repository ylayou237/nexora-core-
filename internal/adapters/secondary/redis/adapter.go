package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Adapter est le point d'entrée unique pour toutes les interactions Redis.
// Il porte la connexion (*redis.Client) et c'est sur LUI que sont rattachées
// les méthodes StartSession, GetByID, etc. (définies dans session_repository.go).
type Adapter struct {
	Client *redis.Client
}

// NewAdapter initialise la connexion Redis.
// Il supporte intelligemment le mode Cluster (Sentinel) ou Simple (Standalone/Miniredis).
func NewAdapter(ctx context.Context, masterName string, addrs []string, password string) (*Adapter, error) {
	if len(addrs) == 0 {
		return nil, fmt.Errorf("redis: aucune adresse fournie")
	}

	var rdb *redis.Client

	// --- LOGIQUE DE SÉLECTION DU MODE ---

	if masterName != "" {
		// MODE SENTINEL (Production Carrier-Grade)
		// Permet le basculement automatique si le maître tombe.
		rdb = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    masterName,
			SentinelAddrs: addrs,
			Password:      password,
			DB:            0,
			PoolSize:      100, // Optimisé pour supporter la charge de 50k users
			MinIdleConns:  10,  // Garde des connexions chaudes
		})
	} else {
		// MODE STANDALONE (Développement / Test / Miniredis)
		rdb = redis.NewClient(&redis.Options{
			Addr:     addrs[0],
			Password: password,
			DB:       0,
			PoolSize: 50, // Suffisant pour le dev
		})
	}

	// --- VÉRIFICATION DE SANTÉ (PING) ---
	// On tente de pinger Redis avec un timeout court.
	// Si ça échoue, on réessaie 3 fois pour éviter les faux négatifs au démarrage.

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var err error
	for i := 0; i < 3; i++ {
		if err = rdb.Ping(pingCtx).Err(); err == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	if err != nil {
		return nil, fmt.Errorf("redis: impossible de se connecter après 3 tentatives: %w", err)
	}

	return &Adapter{Client: rdb}, nil
}

// Close ferme proprement le pool de connexions.
func (a *Adapter) Close() error {
	if a.Client != nil {
		return a.Client.Close()
	}
	return nil
}
