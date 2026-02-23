package ports_test

import (
	"testing"

	"github.com/yvan/nexora-core/internal/adapters/secondary/redis"
	"github.com/yvan/nexora-core/internal/core/ports"
)

func TestInterfaceCompliance(t *testing.T) {
	// Vérifie que le repo de Session est complet (Start, Terminate, Health...)
	var _ ports.SessionRepository = (*redis.SessionRepository)(nil)

	// ✅ Ajoute aussi celui du Refresh Token pour être sûr que tout est vert
	var _ ports.RefreshTokenRepository = (*redis.RedisRefreshTokenRepo)(nil)
}
