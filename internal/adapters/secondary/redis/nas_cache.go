package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/yvan/nexora-core/internal/core/domain"

	"github.com/redis/go-redis/v9"
)

// NasCachePrefix is the Redis key prefix for NAS objects.
const NasCachePrefix = "nas:ip:"

// ErrNasNotFound is returned when a NAS is not found in cache.
var ErrNasNotFound = fmt.Errorf("nas_cache: NAS not found in cache")

// SetNas caches a NAS object in Redis with a TTL.
// Ensures atomic storage and proper serialization.
func (a *Adapter) SetNas(ctx context.Context, nas *domain.NAS, ttl time.Duration) error {
	if nas == nil {
		return fmt.Errorf("nas_cache: cannot cache nil NAS")
	}

	data, err := json.Marshal(nas)
	if err != nil {
		return fmt.Errorf("nas_cache: marshal failed: %w", err)
	}

	key := fmt.Sprintf("%s%s", NasCachePrefix, nas.IP().String())

	if err := a.Client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("nas_cache: redis set failed: %w", err)
	}

	return nil
}

// GetNas retrieves a NAS from Redis by its IP address.
// Returns ErrNasNotFound if the object is missing.
func (a *Adapter) GetNas(ctx context.Context, ip string) (*domain.NAS, error) {
	key := fmt.Sprintf("%s%s", NasCachePrefix, ip)

	val, err := a.Client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrNasNotFound
		}
		return nil, fmt.Errorf("nas_cache: redis get failed: %w", err)
	}

	var nas domain.NAS
	if err := json.Unmarshal([]byte(val), &nas); err != nil {
		return nil, fmt.Errorf("nas_cache: unmarshal failed: %w", err)
	}

	if nas.ID.String() == "" {
		return nil, fmt.Errorf("nas_cache: corrupted data for IP %s", ip)
	}

	return &nas, nil
}

// DeleteNas removes a NAS from Redis manually.
// Should be called after updates in PostgreSQL to maintain cache consistency.
func (a *Adapter) DeleteNas(ctx context.Context, ip string) error {
	key := fmt.Sprintf("%s%s", NasCachePrefix, ip)

	if err := a.Client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("nas_cache: redis delete failed: %w", err)
	}

	return nil
}
