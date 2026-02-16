package ports_test

import (
	"testing"

	// REMARQUE : Vérifie bien que le nom du module dans ton go.mod
	// correspond à "github.com/yvan/nexora-core"
	"github.com/yvan/nexora-core/internal/adapters/secondary/redis"
	"github.com/yvan/nexora-core/internal/core/ports"
)

func TestSessionRepositoryCompliance(t *testing.T) {
	// Cette ligne déclenche l'erreur de build si l'implémentation est incomplète.
	// Elle vérifie StartSession, GetByID, UpdateUsage, TerminateSession et Exists.
	var _ ports.SessionRepository = (*redis.SessionRepository)(nil)
}
