package domain_test

import (
	"testing"

	"github.com/yvan/nexora-core/internal/core/domain"
)

func TestTenantHierarchy(t *testing.T) {
	// 1. Création d'un Operator (root)
	// On met 'false' pour le portail (pas nécessaire pour l'opérateur racine)
	operatorID, _ := domain.NewTenantID("123e4567-e89b-12d3-a456-426614174010")
	operator, err := domain.NewTenant(operatorID, "OperatorRoot", domain.TenantOperator, nil, false) // 👈 Ajout false
	if err != nil {
		t.Fatalf("Failed to create operator tenant: %v", err)
	}
	if !operator.IsRoot() {
		t.Errorf("Operator should be root")
	}

	// 2. Création d'un Provider (enfant de l'Operator)
	// On met 'true' ici pour tester l'activation du portail captif
	providerID, _ := domain.NewTenantID("123e4567-e89b-12d3-a456-426614174011")
	provider, err := domain.NewTenant(providerID, "Provider01", domain.TenantProvider, operator, true) // 👈 Ajout true
	if err != nil {
		t.Fatalf("Failed to create provider tenant: %v", err)
	}
	if provider.IsRoot() {
		t.Errorf("Provider should not be root")
	}
	if provider.ParentID() == nil || provider.ParentID().String() != operator.ID().String() {
		t.Errorf("Provider parentID not set correctly")
	}
	// Vérification de la nouvelle feature Config
	if !provider.IsPortalEnabled() {
		t.Errorf("Provider should have portal enabled")
	}

	// 3. Création d'un Reseller (enfant du Provider)
	// On met 'false'
	resellerID, _ := domain.NewTenantID("123e4567-e89b-12d3-a456-426614174012")
	reseller, err := domain.NewTenant(resellerID, "Reseller01", domain.TenantReseller, provider, false) // 👈 Ajout false
	if err != nil {
		t.Fatalf("Failed to create reseller tenant: %v", err)
	}
	if reseller.IsRoot() {
		t.Errorf("Reseller should not be root")
	}
	if reseller.ParentID() == nil || reseller.ParentID().String() != provider.ID().String() {
		t.Errorf("Reseller parentID not set correctly")
	}

	// 4. Tentative de création invalide (Mauvaise hiérarchie)
	// On doit quand même passer le booléen pour que ça compile
	_, err = domain.NewTenant(resellerID, "InvalidReseller", domain.TenantReseller, operator, false) // 👈 Ajout false
	if err == nil {
		t.Errorf("Expected error when creating Reseller with wrong parent type")
	}
}
