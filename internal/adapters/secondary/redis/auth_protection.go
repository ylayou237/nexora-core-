package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// -----------------------------------------------------------------------------// Config (injectée depuis internal/config)
// -----------------------------------------------------------------------------

type AuthProtectionConfig struct {
	MaxFailedAttempts int
	LockDuration      time.Duration
	WindowDuration    time.Duration
}

// Valeurs par défaut safe (au cas où le wiring passe des zéros)
func (c AuthProtectionConfig) withDefaults() AuthProtectionConfig {
	if c.MaxFailedAttempts <= 0 {
		c.MaxFailedAttempts = 5
	}
	if c.LockDuration <= 0 {
		c.LockDuration = 15 * time.Minute
	}
	if c.WindowDuration <= 0 {
		c.WindowDuration = 1 * time.Hour
	}
	return c
}

// -----------------------------------------------------------------------------// Redis keys
// -----------------------------------------------------------------------------

const authAttemptsKeyFormat = "auth:attempts:%s"

// -----------------------------------------------------------------------------// Repo
// -----------------------------------------------------------------------------

type AuthProtectionRepo struct {
	client redis.UniversalClient
	cfg    AuthProtectionConfig
}

// NewAuthProtectionRepo crée le repo avec un client redis + configuration
func NewAuthProtectionRepo(client redis.UniversalClient, cfg AuthProtectionConfig) *AuthProtectionRepo {
	if client == nil {
		panic("Impossible de créer un AuthProtectionRepo avec un client redis nil")
	}
	return &AuthProtectionRepo{
		client: client,
		cfg:    cfg.withDefaults(),
	}
}

func (r *AuthProtectionRepo) attemptsKey(identity string) string {
	return fmt.Sprintf(authAttemptsKeyFormat, identity)
}

// IsLocked vérifie si l'identité est bloquée et retourne le temps restant (TTL)
func (r *AuthProtectionRepo) IsLocked(ctx context.Context, identity string) (bool, time.Duration, error) {
	key := r.attemptsKey(identity)

	attempts, err := r.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}

	if attempts >= int64(r.cfg.MaxFailedAttempts) {
		ttl, err := r.client.TTL(ctx, key).Result()
		if err != nil {
			return true, 0, err
		}
		// ttl peut être -1 si pas d'expiration (ne devrait pas arriver)
		if ttl < 0 {
			return true, r.cfg.LockDuration, nil
		}
		return true, ttl, nil
	}

	return false, 0, nil
}

// RecordFailedAttempt ajoute +1 aux échecs et gère les expirations
func (r *AuthProtectionRepo) RecordFailedAttempt(ctx context.Context, identity string) (int64, error) {
	key := r.attemptsKey(identity)

	pipe := r.client.TxPipeline()
	incr := pipe.Incr(ctx, key)
	ttlCmd := pipe.TTL(ctx, key)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	attempts := incr.Val()
	currentTTL := ttlCmd.Val()

	// Si première tentative ou pas de TTL -> fenêtre standard
	if attempts == 1 || currentTTL < 0 {
		_ = r.client.Expire(ctx, key, r.cfg.WindowDuration).Err()
	}

	// Si on atteint le seuil, on applique le verrouillage strict
	if attempts >= int64(r.cfg.MaxFailedAttempts) {
		_ = r.client.Expire(ctx, key, r.cfg.LockDuration).Err()
	}

	return attempts, nil
}

// ClearAttempts remet le compteur à zéro en cas de succès,
// MAIS ne doit pas annuler un lock déjà actif.
func (r *AuthProtectionRepo) ClearAttempts(ctx context.Context, identity string) error {
	key := r.attemptsKey(identity)

	attempts, err := r.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return err
	}

	// Si déjà locké, ne pas supprimer la clé (sinon bypass)
	if attempts >= int64(r.cfg.MaxFailedAttempts) {
		return nil
	}

	return r.client.Del(ctx, key).Err()
}
