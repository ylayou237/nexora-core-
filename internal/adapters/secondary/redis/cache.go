package redis

import (
	"context"
	"time"
)

// CacheRepository implémente ports.CacheRepository
type CacheRepository struct {
	adapter *Adapter
}

func NewCacheRepository(a *Adapter) *CacheRepository {
	return &CacheRepository{adapter: a}
}

// ✅ CORRECTION : Les méthodes doivent être sur (r *CacheRepository)
// On utilise r.adapter.Client pour accéder à Redis.

// Set stocke une valeur brute avec un TTL.
func (r *CacheRepository) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return r.adapter.Client.Set(ctx, key, value, ttl).Err()
}

// Get récupère une valeur brute.
func (r *CacheRepository) Get(ctx context.Context, key string) (string, error) {
	return r.adapter.Client.Get(ctx, key).Result()
}

// Delete supprime une clé.
func (r *CacheRepository) Delete(ctx context.Context, key string) error {
	return r.adapter.Client.Del(ctx, key).Err()
}

// SetNX stocke la valeur uniquement si la clé n'existe pas.
func (r *CacheRepository) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	return r.adapter.Client.SetNX(ctx, key, value, ttl).Result()
}
