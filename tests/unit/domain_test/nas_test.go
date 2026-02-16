package domain_test

import (
	"net"
	"testing"

	"github.com/yvan/nexora-core/internal/core/domain"
)

func TestNASDefensiveCopy(t *testing.T) {
	clock := NewFakeClock()

	nasID, _ := domain.NewNasID("nas_core_01")
	tenantID, _ := domain.NewTenantID("11111111-1111-4111-8111-111111111111")

	// IP Source
	ipBytes := []byte{192, 168, 1, 10}
	originalIP := net.IP(ipBytes)

	nas, err := domain.NewNAS(
		nasID,
		tenantID,
		"nas-id-radius",
		"MikroTik Core",
		originalIP,
		"s3cr3t",
		clock,
	)
	if err != nil {
		t.Fatalf("NAS creation failed: %v", err)
	}

	// TEST CRITIQUE : Modifier l'IP originale ne doit PAS affecter le NAS
	// C'est ce qu'on appelle la copie défensive
	originalIP[3] = 99 // On change .10 en .99

	nasIP := nas.IP()
	if nasIP[3] == 99 {
		t.Error("Security Breach: NAS IP was modified by external reference!")
	}
	if nasIP[3] != 10 {
		t.Error("NAS IP corrupted")
	}
}
