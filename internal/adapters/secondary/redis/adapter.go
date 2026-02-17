package redis

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Adapter struct {
	// Utilisation de l'interface UniversalClient pour accepter
	// aussi bien les clients simples que TLS ou Failover
	Client redis.UniversalClient
}

func NewAdapter(ctx context.Context, password string, addrs []string, masterName string) (*Adapter, error) {
	if len(addrs) == 0 {
		return nil, fmt.Errorf("redis: aucune adresse fournie")
	}

	var rdb redis.UniversalClient

	if masterName != "" {
		// MODE SENTINEL
		rdb = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    masterName,
			SentinelAddrs: addrs,
			Password:      password,
		})
	} else {
		// MODE STANDALONE (Upstash nécessite absolument TLSConfig pour éviter l'EOF)
		rdb = redis.NewClient(&redis.Options{
			Addr:     addrs[0],
			Password: password,
			DB:       0,
			// ✅ CRITIQUE : Cette ligne résout l'erreur EOF sur Upstash
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		})
	}

	// Test de connexion
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := rdb.Ping(pingCtx).Err(); err != nil {
		return nil, fmt.Errorf("redis: connexion échouée: %w", err)
	}

	return &Adapter{Client: rdb}, nil
}

func (a *Adapter) Close() error {
	if a.Client != nil {
		return a.Client.Close()
	}
	return nil
}
