package domain_test

import (
	"net"
	"testing"

	"github.com/yvan/nexora-core/internal/core/domain"
)

func TestNASDefensiveCopy(t *testing.T) {
	// 1. Initialisation
	clock := NewFakeClock()

	nasID, _ := domain.NewNasID("nas_core_01")
	tenantID, _ := domain.NewTenantID("11111111-1111-4111-8111-111111111111")

	// 2. IP Source
	ipBytes := []byte{192, 168, 1, 10}
	originalIP := net.IP(ipBytes)

	// ✅ Correction : Utilisation du struct NewNASParams pour satisfaire SonarLint et le compilateur
	nas, err := domain.NewNAS(domain.NewNASParams{
		ID:         nasID,
		TenantID:   tenantID,
		Identifier: "nas-id-radius",
		ShortName:  "MikroTik Core",
		IP:         originalIP,
		Secret:     "s3cr3t",
	}, clock)

	if err != nil {
		t.Fatalf("NAS creation failed: %v", err)
	}

	// 3. TEST CRITIQUE : Copie défensive
	// On modifie l'IP originale après la création du NAS
	originalIP[3] = 99 // On tente de changer .10 en .99 par référence

	nasIP := nas.IP()

	t.Run("Vérification de l'isolation mémoire", func(t *testing.T) {
		if nasIP[3] == 99 {
			t.Error("Security Breach: NAS IP was modified by external reference! Defensive copy failed.")
		}
		if nasIP[3] != 10 {
			t.Errorf("NAS IP corrupted: expected 10, got %d", nasIP[3])
		}
	})

	t.Run("Vérification des getters", func(t *testing.T) {
		if nas.ShortName != "MikroTik Core" {
			t.Errorf("Expected MikroTik Core, got %s", nas.ShortName)
		}
	})
}
