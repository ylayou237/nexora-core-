package services

import (
	"context"
	"time"

	"github.com/yvan/nexora-core/internal/core/domain"
	"github.com/yvan/nexora-core/internal/core/ports" // Utilise l'interface
)

type CacheService struct {
	repo ports.CacheRepository
}

func NewCacheService(repo ports.CacheRepository) *CacheService {
	return &CacheService{repo: repo}
}

// StoreSession lie un JTI à un utilisateur (utile pour le "Logout all devices")
func (s *CacheService) StoreSession(ctx context.Context, jti string, userID domain.UserID, ttl time.Duration) error {
	key := "session:" + jti
	return s.repo.Set(ctx, key, userID.String(), ttl)
}

// BlacklistToken révoque un token avec un TTL intelligent
func (s *CacheService) BlacklistToken(ctx context.Context, jti string, remainingLifetime time.Duration) error {
	key := "blacklist:" + jti
	// On ne stocke le flag que pour la durée de vie restante du JWT
	return s.repo.Set(ctx, key, "1", remainingLifetime)
}

// IsTokenBlacklisted vérifie la validité (utilisé par le middleware)
func (s *CacheService) IsTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	key := "blacklist:" + jti
	_, err := s.repo.Get(ctx, key)
	if err != nil {
		// En Go-Redis, Nil signifie que la clé n'existe pas (donc pas en blacklist)
		return false, nil
	}
	return true, nil
}
