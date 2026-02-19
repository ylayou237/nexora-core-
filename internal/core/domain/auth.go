package domain

import (
	"context"
	"time"
)

// AuthProtectionRepository définit le contrat pour le bouclier anti-brute force
type AuthProtectionRepository interface {
	IsLocked(ctx context.Context, identifier string) (bool, time.Duration, error)
	RecordFailedAttempt(ctx context.Context, identifier string) (int64, error)
	ClearAttempts(ctx context.Context, identifier string) error
}
