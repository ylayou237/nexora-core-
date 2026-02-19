package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Constantes configurables
const (
	MaxFailedAttempts     = 5
	LockDuration          = 15 * time.Minute
	WindowDuration        = 1 * time.Hour
	authAttemptsKeyFormat = "auth:attempts:%s"
)

type AuthProtectionRepo struct {
	client redis.UniversalClient
}

// ✅ CORRECTION : Le constructeur accepte maintenant redis.UniversalClient
// Cela permet de passer redisAdapter.Client depuis le main.go
func NewAuthProtectionRepo(client redis.UniversalClient) *AuthProtectionRepo {
	if client == nil {
		panic("Impossible de créer un AuthProtectionRepo avec un client redis nil")
	}
	return &AuthProtectionRepo{client: client}
}

// IsLocked vérifie si l'utilisateur est bloqué et retourne le temps restant
func (r *AuthProtectionRepo) IsLocked(ctx context.Context, email string) (bool, time.Duration, error) {
	key := fmt.Sprintf(authAttemptsKeyFormat, email)

	attempts, err := r.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return false, 0, nil
	} else if err != nil {
		return false, 0, err
	}

	if attempts >= MaxFailedAttempts {
		ttl, err := r.client.TTL(ctx, key).Result()
		if err != nil {
			return true, 0, err
		}
		return true, ttl, nil
	}

	return false, 0, nil
}

// RecordFailedAttempt ajoute +1 aux échecs et gère les expirations
func (r *AuthProtectionRepo) RecordFailedAttempt(ctx context.Context, email string) (int64, error) {
	key := fmt.Sprintf(authAttemptsKeyFormat, email)

	// Utilisation d'un pipeline pour l'atomicité relative
	pipe := r.client.Pipeline()
	incr := pipe.Incr(ctx, key)
	ttlCmd := pipe.TTL(ctx, key)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	attempts := incr.Val()
	currentTTL := ttlCmd.Val()

	// Si c'est la première tentative ou si la clé n'avait pas de TTL (-1)
	if attempts == 1 || currentTTL < 0 {
		r.client.Expire(ctx, key, WindowDuration)
	}

	// Si on atteint le seuil, on applique le verrouillage strict
	if attempts >= MaxFailedAttempts {
		r.client.Expire(ctx, key, LockDuration)
	}

	return attempts, nil
}

// ClearAttempts remet les compteurs à zéro en cas de succès
func (r *AuthProtectionRepo) ClearAttempts(ctx context.Context, email string) error {
	key := fmt.Sprintf(authAttemptsKeyFormat, email)
	return r.client.Del(ctx, key).Err()
}
