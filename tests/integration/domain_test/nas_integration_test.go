package domain_test

import (
	"net"
	"testing"
	"time"

	"github.com/yvan/nexora-core/internal/core/domain"
)

func TestNASLifecycleAndVersioning(t *testing.T) {
	// Setup avec temps fixe
	fixedTime := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)
	clock := &FakeClock{now: fixedTime}

	tenantID, _ := domain.NewTenantID("123e4567-e89b-12d3-a456-426614174000")
	nasID, _ := domain.NewNasID("nas-001")
	ip := net.ParseIP("192.168.1.1")

	// ✅ Correction : Passage aux paramètres structurés (NewNASParams)
	nas, err := domain.NewNAS(domain.NewNASParams{
		ID:         nasID,
		TenantID:   tenantID,
		Identifier: "nas-identifier-01",
		ShortName:  "mikrotik-01",
		IP:         ip,
		Secret:     "secret123",
	}, clock)

	if err != nil {
		t.Fatalf("Failed to create NAS: %v", err)
	}

	// VÉRIFICATION : Accès aux champs exportés (Standardisé avec le reste du projet)
	if nas.Version != 1 {
		t.Errorf("Expected initial version 1, got %d", nas.Version)
	}

	// Test Deactivation
	nas.Deactivate(clock)
	if nas.Active {
		t.Errorf("NAS should be inactive")
	}

	// Test Versioning après mutation (bumpVersion interne)
	if nas.Version != 2 {
		t.Errorf("Version should be 2 after deactivation, got %d", nas.Version)
	}

	// Test Time Update
	if !nas.UpdatedAt.Equal(fixedTime) {
		t.Errorf("UpdatedAt should match clock time, got %v", nas.UpdatedAt)
	}
}
