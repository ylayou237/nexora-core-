package domain

import (
	"errors"
)

// --- TenantType Enum ---
type TenantType string

const (
	TenantOperator TenantType = "operator"
	TenantProvider TenantType = "provider"
	TenantReseller TenantType = "reseller"
)

// --- Value Object: TenantConfig ---
// Regroupe les réglages techniques modifiables.
type TenantConfig struct {
	PortalEnabled bool
	// Futur : Timezone string, Currency string, LogoURL string...
}

// --- Aggregate Root: Tenant ---

type Tenant struct {
	id       TenantID
	parentID *TenantID // Nil pour Operator
	name     string
	typ      TenantType

	// Configuration (Ajout V5)
	config TenantConfig
}

// --- Factory ---

func NewTenant(
	id TenantID,
	name string,
	typ TenantType,
	parent *Tenant,
	portalEnabled bool, // 👈 Nouvel argument requis
) (*Tenant, error) {

	if name == "" {
		return nil, errors.New("tenant name cannot be empty")
	}

	var pID *TenantID

	// Logique de validation hiérarchique (Gardée de la version B)
	switch typ {
	case TenantOperator:
		if parent != nil {
			return nil, errors.New("operator must be root (no parent)")
		}
		pID = nil
	case TenantProvider:
		if parent == nil {
			return nil, errors.New("provider must have a parent")
		}
		if parent.Type() != TenantOperator {
			return nil, ErrInvalidHierarchy
		}
		idCopy := parent.ID()
		pID = &idCopy
	case TenantReseller:
		if parent == nil {
			return nil, errors.New("reseller must have a parent")
		}
		if parent.Type() != TenantProvider {
			return nil, ErrInvalidHierarchy
		}
		idCopy := parent.ID()
		pID = &idCopy
	default:
		return nil, ErrInvalidTenantType
	}

	return &Tenant{
		id:       id,
		parentID: pID,
		name:     name,
		typ:      typ,
		config: TenantConfig{
			PortalEnabled: portalEnabled,
		},
	}, nil
}

// --- Rehydration ---

func RehydrateTenant(
	id TenantID,
	parentID *TenantID,
	name string,
	typ TenantType,
	portalEnabled bool,
) *Tenant {
	return &Tenant{
		id:       id,
		parentID: parentID,
		name:     name,
		typ:      typ,
		config: TenantConfig{
			PortalEnabled: portalEnabled,
		},
	}
}

// --- Getters ---

func (t *Tenant) ID() TenantID { return t.id }
func (t *Tenant) ParentID() *TenantID {
	if t.parentID == nil {
		return nil
	}
	copy := *t.parentID
	return &copy
}
func (t *Tenant) Name() string     { return t.name }
func (t *Tenant) Type() TenantType { return t.typ }

// IsRoot returns true if this tenant is an Operator
func (t *Tenant) IsRoot() bool {
	return t.parentID == nil
}

// --- Config Methods ---

func (t *Tenant) IsPortalEnabled() bool {
	return t.config.PortalEnabled
}

func (t *Tenant) EnablePortal() {
	t.config.PortalEnabled = true
}

func (t *Tenant) DisablePortal() {
	t.config.PortalEnabled = false
}

const SystemTenantID = "00000000-0000-0000-0000-000000000000"

func SystemTenant() TenantID {
	return TenantID(SystemTenantID)
}
