package domain_test

import (
	"testing"
	"time"

	"github.com/yvan/nexora-core/internal/core/domain"
)

func TestPlanUpdates(t *testing.T) {
	// 1. Initialisation de l'horloge de test
	clock := NewFakeClock()

	// 2. Préparation des IDs de domaine
	planID, _ := domain.NewPlanID("plan_silver")
	tenantID, _ := domain.NewTenantID("11111111-1111-4111-8111-111111111111")

	// 3. Création du Plan via le Parameter Object (Correction Sonar S107)
	plan, err := domain.NewPlan(domain.NewPlanParams{
		ID:          planID,
		TenantID:    tenantID,
		Name:        "Silver Plan",
		Description: "For testing",
		DataQuota:   500 * 1024 * 1024, // 500 MB
		MaxUpload:   10000,             // bps
		MaxDownload: 50000,             // bps
		MaxSessions: 2,
		TimeQuota:   30 * 24 * time.Hour,
	}, clock)

	if err != nil {
		t.Fatalf("Plan creation failed: %v", err)
	}

	// Avancer l'horloge pour tester le bump de UpdatedAt
	clock.Advance(1 * time.Hour)

	// 4. Test de la mise à jour des limites (UpdateLimits)
	// Note : La signature de UpdateLimits n'a pas été changée car elle reste sous le seuil Sonar.
	err = plan.UpdateLimits(
		1024*1024*1024, // 1 GB
		20000,          // New MaxUpload
		100000,         // New MaxDownload
		3,              // Max Sessions incrémenté
		30*24*time.Hour,
		clock,
	)

	if err != nil {
		t.Errorf("UpdateLimits failed: %v", err)
	}

	// 5. Assertions
	t.Run("Vérification des valeurs mises à jour", func(t *testing.T) {
		if plan.MaxSessions() != 3 {
			t.Errorf("Expected 3 sessions, got %d", plan.MaxSessions())
		}
		if plan.DataQuota() != 1024*1024*1024 {
			t.Errorf("Expected 1GB quota, got %d", plan.DataQuota())
		}
		if plan.MaxDownload() != 100000 {
			t.Errorf("Expected 100000 bps download, got %d", plan.MaxDownload())
		}
	})

	t.Run("Vérification des métadonnées", func(t *testing.T) {
		// La version doit être passée à 2 après UpdateLimits
		if plan.Version() != 2 {
			t.Errorf("Expected version 2, got %d", plan.Version())
		}
		// La date de mise à jour doit avoir avancé d'une heure
		if !plan.UpdatedAt().After(plan.CreatedAt()) {
			t.Error("Expected UpdatedAt to be after CreatedAt")
		}
	})
}
