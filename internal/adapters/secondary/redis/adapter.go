package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Adapter gère la connexion à Redis avec support HA via Sentinel.
type Adapter struct {
	Client *redis.Client
}

// NewAdapter initialise le client Redis.
// Supporte soit un mode Standalone (dev) soit Sentinel (prod Cloud).
func NewAdapter(ctx context.Context, masterName string, addrs []string, password string) (*Adapter, error) {
	if len(addrs) == 0 {
		return nil, fmt.Errorf("adresse Redis non fournie")
	}

	var rdb *redis.Client

	if masterName != "" {
		// MODE SENTINEL (Production Carrier-Grade)
		rdb = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    masterName,
			SentinelAddrs: addrs,
			Password:      password,
			DB:            0, // DB par défaut
		})
	} else {
		// MODE STANDALONE (Développement local)
		rdb = redis.NewClient(&redis.Options{
			Addr:     addrs[0],
			Password: password,
			DB:       0,
		})
	}

	// Test de connexion avec timeout et retry simple
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var pingErr error
	for i := 0; i < 3; i++ {
		pingErr = rdb.Ping(pingCtx).Err()
		if pingErr == nil {
			break
		}
		time.Sleep(500 * time.Millisecond) // retry backoff simple
	}

	if pingErr != nil {
		return nil, fmt.Errorf("impossible de joindre Redis: %w", pingErr)
	}

	return &Adapter{Client: rdb}, nil
}

// Close ferme proprement les connexions au cache.
func (a *Adapter) Close() error {
	if a.Client == nil {
		return nil
	}
	return a.Client.Close()
}
