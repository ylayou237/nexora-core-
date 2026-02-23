package domain_test

import (
	"testing"
	"time"

	"github.com/yvan/nexora-core/internal/core/domain"
)

func TestPlanIntegration(t *testing.T) {
	// 1. Initialisation de l'horloge de test
	clock := &FakeClock{now: time.Now()}

	// 2. Préparation des IDs (Types forts Nexora)
	tenantID, _ := domain.NewTenantID("123e4567-e89b-12d3-a456-426614174000")
	planID, _ := domain.NewPlanID("plan-int-001")

	// 3. ✅ Correction : Utilisation du struct NewPlanParams (Anti-S107 Sonar)
	// On passe de 10 arguments à seulement 2 (Params + Clock)
	plan, err := domain.NewPlan(domain.NewPlanParams{
		ID:          planID,
		TenantID:    tenantID,
		Name:        "Integration Plan",
		Description: "Test Plan Integration",
		DataQuota:   1024 * 1024,
		MaxUpload:   1000,
		MaxDownload: 2000,
		MaxSessions: 5,
		TimeQuota:   3600 * time.Second,
	}, clock)

	if err != nil {
		t.Fatalf("Failed to create plan: %v", err)
	}

	// 4. Vérification initiale
	if plan.MaxSessions() != 5 {
		t.Errorf("Expected maxSessions=5, got %d", plan.MaxSessions())
	}

	// 5. Test de mise à jour des limites
	// Note : La méthode UpdateLimits conserve sa signature originale car elle
	// reste sous le seuil de complexité Sonar.
	err = plan.UpdateLimits(2048*1024, 2000, 4000, 10, 7200*time.Second, clock)

	if err != nil {
		t.Fatalf("Failed to update plan limits: %v", err)
	}

	// 6. Assertions finales
	t.Run("Validation des nouvelles limites", func(t *testing.T) {
		if plan.MaxSessions() != 10 {
			t.Errorf("Expected 10 sessions, got %d", plan.MaxSessions())
		}
		if plan.DataQuota() != 2048*1024 {
			t.Errorf("Expected 2048*1024 data quota, got %d", plan.DataQuota())
		}
	})
}
