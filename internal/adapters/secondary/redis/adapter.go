package redis

import (
	"context"
	"crypto/tls"
	"fmt"

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

	// Configuration optimisée pour Upstash
	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    addrs,
		Username: "default", // Crucial pour Upstash
		Password: password,
		DB:       0,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	})

	// Le Ping pour valider la tuyauterie
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &Adapter{Client: rdb}, nil
}

// Et n'oublie pas la méthode Close pour corriger l'erreur de compilation précédente
func (a *Adapter) Close() error {
	if a.Client != nil {
		return a.Client.Close()
	}
	return nil
}
func (a *Adapter) Ping(ctx context.Context) (string, error) {
	// Si tu utilises go-redis
	pong, err := a.Client.Ping(ctx).Result()
	if err != nil {
		return "", err
	}
	return pong, nil
}
