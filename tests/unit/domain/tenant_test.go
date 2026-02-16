package domain_test

import (
	"testing"

	"github.com/yvan/nexora-core/internal/core/domain"
)

func TestTenantHierarchy(t *testing.T) {
	// UUIDs valides (RFC 4122)
	opID, _ := domain.NewTenantID("11111111-1111-4111-8111-111111111111")
	provID, _ := domain.NewTenantID("22222222-2222-4222-8222-222222222222")
	resID, _ := domain.NewTenantID("33333333-3333-4333-8333-333333333333")

	// 1. TEST : Operator (Root) - Doit réussir sans parent
	// On ajoute 'false' pour portalEnabled
	operator, err := domain.NewTenant(opID, "Orange", domain.TenantOperator, nil, false)
	if err != nil {
		t.Fatalf("Failed to create operator: %v", err)
	}

	// 2. TEST : Provider (Enfant d'Operator) - Doit réussir
	// On active le portail ici pour tester la config
	provider, err := domain.NewTenant(provID, "IspConnect", domain.TenantProvider, operator, true)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	if !provider.IsPortalEnabled() {
		t.Error("Provider portal should be enabled")
	}

	// 3. TEST : Reseller (Enfant de Provider) - Doit réussir
	_, err = domain.NewTenant(resID, "LocalReseller", domain.TenantReseller, provider, false)
	if err != nil {
		t.Fatalf("Failed to create reseller: %v", err)
	}

	// 4. TEST DE SÉCURITÉ : Provider sans parent (Orphelin)
	// Un Provider ne peut pas exister sans être rattaché à un Operator.
	_, err = domain.NewTenant(provID, "Orphan Provider", domain.TenantProvider, nil, false)
	if err == nil {
		t.Error("Should fail to create provider without parent")
	}

	// 5. TEST DE RÈGLE MÉTIER : Saut de niveau hiérarchique
	// Un Reseller doit obligatoirement être rattaché à un Provider (pas directement à l'Operator).
	_, err = domain.NewTenant(resID, "Bad Hierarchy", domain.TenantReseller, operator, false)
	if err == nil {
		t.Error("Should fail to create reseller attached directly to operator")
	}
}
