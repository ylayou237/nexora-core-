package domain

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// AuthProtectionRepository définit le contrat pour le bouclier anti-brute force
type AuthProtectionRepository interface {
	IsLocked(ctx context.Context, identifier string) (bool, time.Duration, error)
	RecordFailedAttempt(ctx context.Context, identifier string) (int64, error)
	ClearAttempts(ctx context.Context, identifier string) error
}

func GenerateSecureRandomString(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		// Dans un système critique, si le générateur de nombres aléatoires du
		// système d'exploitation échoue, on doit stopper l'exécution.
		panic(fmt.Sprintf("critical: failed to generate secure random bytes: %v", err))
	}
	return hex.EncodeToString(b)
}
