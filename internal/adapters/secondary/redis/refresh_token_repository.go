package redis

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yvan/nexora-core/internal/core/domain"
	"github.com/yvan/nexora-core/internal/core/ports"
)

// Constantes pour le namespace Redis (Évite les collisions et les typos)
const (
	refreshTokenKeyPrefix = "rt:%s"     // rt = refresh_token
	tokenFamilyKeyPrefix  = "family:%s" // family = groupe de tokens par appareil
)

//go:embed scripts/rotate_token.lua
var rotateTokenLua string

type RedisRefreshTokenRepo struct {
	client       *redis.Client
	rotateScript *redis.Script
}

// NewRedisRefreshTokenRepo initialise le repository avec le script Lua pré-chargé
func NewRedisRefreshTokenRepo(client *redis.Client) ports.RefreshTokenRepository {
	return &RedisRefreshTokenRepo{
		client:       client,
		rotateScript: redis.NewScript(rotateTokenLua),
	}
}

// Health implémente le check de santé requis par AuthService.CheckIntegrity
func (r *RedisRefreshTokenRepo) Health(ctx context.Context) error {
	if err := r.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis refresh store unreachable: %w", err)
	}
	return nil
}

// Save : Stockage initial d'un nouveau token
// Save : Stockage initial d'un nouveau token avec hachage consistant
func (r *RedisRefreshTokenRepo) Save(ctx context.Context, t *domain.RefreshToken) error {
	// Génération de la clé à partir du secret brut pour garantir la cohérence avec Rotate
	actualHash := r.hash(t.TokenRaw)
	key := fmt.Sprintf(refreshTokenKeyPrefix, actualHash)
	familyKey := fmt.Sprintf(tokenFamilyKeyPrefix, t.FamilyID)

	data, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("token_repo: fail to marshal token: %w", err)
	}

	ttl := time.Until(t.ExpiresAt)
	if ttl <= 0 {
		return fmt.Errorf("token_repo: token already expired")
	}

	// Utilisation d'un pipeline pour garantir l'atomicité et réduire les aller-retours réseau
	pipe := r.client.Pipeline()
	pipe.Set(ctx, key, data, ttl)
	pipe.SAdd(ctx, familyKey, key)
	pipe.PExpire(ctx, familyKey, ttl) // La famille expire avec son dernier membre

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("token_repo: fail to execute redis pipeline: %w", err)
	}

	return nil
}

// Rotate : Exécute la rotation atomique sécurisée via Lua (Détection de rejeu incluse)
func (r *RedisRefreshTokenRepo) Rotate(
	ctx context.Context,
	oldTokenID domain.RefreshTokenID,
	now time.Time,
	ip, ua, device string,
) (*domain.RefreshToken, error) {

	// 1. Calcul du hash pour identifier le token existant dans Redis
	oldHash := r.hash(string(oldTokenID))
	oldKey := fmt.Sprintf(refreshTokenKeyPrefix, oldHash)

	// 2. Préparation du nouveau secret brut (qui sera renvoyé au client)
	newRaw := domain.GenerateSecureRandomString(32)
	newHash := r.hash(newRaw)
	newKey := fmt.Sprintf(refreshTokenKeyPrefix, newHash)

	// 3. Payload partiel pour Lua (le script complètera avec l'identité)
	newToken := &domain.RefreshToken{
		ID:        domain.RefreshTokenID(newHash),
		CreatedAt: now,
		ExpiresAt: now.Add(7 * 24 * time.Hour), // ✅ FIX : Durée locale (7 jours) au lieu de s.config
		TokenRaw:  newRaw,
	}

	data, err := json.Marshal(newToken)
	if err != nil {
		return nil, fmt.Errorf("token_repo: fail to marshal new token: %w", err)
	}

	ttlMs := int64(time.Until(newToken.ExpiresAt).Milliseconds())

	// 4. Exécution du script Lua atomique
	keys := []string{oldKey, newKey}
	result, err := r.rotateScript.Run(ctx, r.client, keys, data, ttlMs, now.Unix()).Result()

	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "REPLAY_DETECTED") {
			return nil, domain.ErrReplayDetected
		}
		if strings.Contains(errStr, "TOKEN_NOT_FOUND") {
			return nil, fmt.Errorf("token not found in redis")
		}
		return nil, fmt.Errorf("token_repo: lua execution error: %w", err)
	}

	// 5. Décodage sécurisé de la réponse Lua (Multi-type support Upstash/Redis)
	var resultStr string
	switch v := result.(type) {
	case string:
		resultStr = v
	case []byte:
		resultStr = string(v)
	default:
		return nil, fmt.Errorf("token_repo: unexpected lua return type: %T", result)
	}

	// 6. Reconstruction de l'objet domaine final
	var finalToken domain.RefreshToken
	if err := json.Unmarshal([]byte(resultStr), &finalToken); err != nil {
		return nil, fmt.Errorf("token_repo: fail to unmarshal rotated token: %w", err)
	}

	// 7. Réinjection du secret en clair pour que le client reçoive l'UUID non haché
	finalToken.TokenRaw = newRaw

	return &finalToken, nil
}

// GetByHash : Récupération d'un token via son hash (utilisé pour validation)
func (r *RedisRefreshTokenRepo) GetByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	key := fmt.Sprintf(refreshTokenKeyPrefix, hash)

	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("token not found")
		}
		return nil, err
	}

	var t domain.RefreshToken
	err = json.Unmarshal([]byte(val), &t)
	return &t, err
}

// Revoke : Logout sécurisé. Supprime le token ET toute sa lignée (famille)
func (r *RedisRefreshTokenRepo) Revoke(ctx context.Context, id domain.RefreshTokenID) error {
	rawStr := string(id)
	if rawStr == "" {
		return nil
	}

	// 1. 🛡️ IMPORTANT : On hache le secret brut pour trouver la clé
	hash := r.hash(rawStr)
	key := fmt.Sprintf(refreshTokenKeyPrefix, hash)

	// 2. On récupère le token pour obtenir son FamilyID
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil // Déjà expiré ou supprimé
		}
		return fmt.Errorf("redis_revoke: %w", err)
	}

	var t domain.RefreshToken
	if err := json.Unmarshal([]byte(val), &t); err != nil {
		return fmt.Errorf("redis_revoke: unmarshal failed: %w", err)
	}

	// 3. Appel de la suppression de famille pour un nettoyage complet
	return r.RevokeFamily(ctx, t.FamilyID, time.Now(), "logout")
}

// RevokeFamily : Supprime atomiquement tous les membres d'une famille
func (r *RedisRefreshTokenRepo) RevokeFamily(ctx context.Context, fID domain.TokenFamilyID, _ time.Time, _ string) error {
	familyKey := fmt.Sprintf(tokenFamilyKeyPrefix, fID)

	// Récupère tous les hashs de tokens enregistrés dans cette famille
	hashes, err := r.client.SMembers(ctx, familyKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil
		}
		return fmt.Errorf("redis_revoke_family: smembers failed: %w", err)
	}

	pipe := r.client.Pipeline()

	// On supprime chaque Refresh Token individuellement
	if len(hashes) > 0 {
		// En Go redis, Del accepte une variadic de strings
		pipe.Del(ctx, hashes...)
	}

	// On supprime l'index de famille
	pipe.Del(ctx, familyKey)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("redis_revoke_family: pipeline failed: %w", err)
	}

	return nil
}

// hash : Helper interne pour le hashage SHA256 des tokens
func (r *RedisRefreshTokenRepo) hash(token string) string {
	h := sha256.New()
	h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil))
}
