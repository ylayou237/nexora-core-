package ports

import (
	"context"
	"time"
)

// SharedJWKSState est l'état partagé (stockage distribué) du JWKS.
// Keys : KID -> private key PEM (déjà chiffrée côté store si besoin).
type SharedJWKSState struct {
	Version    int               `json:"version"`
	CurrentKID string            `json:"current_kid"`
	Keys       map[string]string `json:"keys"`
	CreatedAt  map[string]string `json:"created_at"`
}

type JWKSStore interface {
	TryAcquireLock(ctx context.Context, lockName string, ttl time.Duration) (acquired bool, token string, err error)
	ReleaseLock(ctx context.Context, lockName, token string) (released bool, err error)

	SaveState(ctx context.Context, state SharedJWKSState) error
	LoadState(ctx context.Context) (*SharedJWKSState, error)
}
