package domain

import (
	"errors"
	"net"
	"time"
)

// --- Aggregate Root: NAS ---
// Le NAS (Network Access Server) représente l'équipement réseau (MikroTik, Ubiquiti, etc.)
// qui interagit avec notre serveur RADIUS.

type NAS struct {
	// Champs exportés pour permettre la sérialisation JSON (Redis/API)
	ID         NasID    `json:"id"`
	TenantID   TenantID `json:"tenant_id"`
	Identifier string   `json:"identifier"`
	ShortName  string   `json:"short_name"`
	IPAddress  net.IP   `json:"ip_address"` // Public
	Secret     string   `json:"secret"`
	Active     bool     `json:"active"`

	// Metadata
	Version   uint64    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// --- Parameter Objects (Pour éviter l'erreur SonarQube S107 des > 7 arguments) ---

// NewNASParams contient les données requises pour créer un nouveau NAS
type NewNASParams struct {
	ID         NasID
	TenantID   TenantID
	Identifier string
	ShortName  string
	IP         net.IP
	Secret     string
}

// NASSnapshot contient toutes les données brutes pour reconstruire l'objet
type NASSnapshot struct {
	ID         NasID
	TenantID   TenantID
	Identifier string
	ShortName  string
	IPAddress  net.IP
	Secret     string
	Active     bool
	Version    uint64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// --- Factory (Constructeur) ---

// NewNAS accepte maintenant une struct de params + l'horloge
func NewNAS(params NewNASParams, clock Clock) (*NAS, error) {
	if params.ShortName == "" {
		return nil, errors.New("nas short name cannot be empty")
	}
	if params.IP == nil || params.IP.IsUnspecified() {
		return nil, errors.New("invalid nas ip address")
	}
	if params.Secret == "" {
		return nil, errors.New("radius secret cannot be empty")
	}

	// Copie défensive
	ipCopy := make(net.IP, len(params.IP))
	copy(ipCopy, params.IP)

	now := clock.Now()

	return &NAS{
		ID:         params.ID,
		TenantID:   params.TenantID,
		Identifier: params.Identifier,
		ShortName:  params.ShortName,
		IPAddress:  ipCopy,
		Secret:     params.Secret,
		Active:     true,
		Version:    1,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// --- Rehydration ---

// RehydrateNAS utilise maintenant un Snapshot pour ne prendre qu'un seul argument
func RehydrateNAS(data NASSnapshot) *NAS {
	return &NAS{
		ID:         data.ID,
		TenantID:   data.TenantID,
		Identifier: data.Identifier,
		ShortName:  data.ShortName,
		IPAddress:  data.IPAddress,
		Secret:     data.Secret,
		Active:     data.Active,
		Version:    data.Version,
		CreatedAt:  data.CreatedAt,
		UpdatedAt:  data.UpdatedAt,
	}
}

// --- Logique Métier ---

func (n *NAS) Activate(clock Clock) {
	if n.Active {
		return
	}
	n.Active = true
	n.bumpVersion(clock)
}

func (n *NAS) Deactivate(clock Clock) {
	if !n.Active {
		return
	}
	n.Active = false
	n.bumpVersion(clock)
}

func (n *NAS) UpdateSecret(newSecret string, clock Clock) error {
	if newSecret == "" {
		return errors.New("secret cannot be empty")
	}
	n.Secret = newSecret
	n.bumpVersion(clock)
	return nil
}

func (n *NAS) bumpVersion(clock Clock) {
	n.Version++
	n.UpdatedAt = clock.Now()
}

// --- Getters Spécifiques ---

// IP retourne une copie de l'IP pour protéger l'intégrité de l'agrégat.
func (n *NAS) IP() net.IP {
	if n.IPAddress == nil {
		return nil
	}
	out := make(net.IP, len(n.IPAddress))
	copy(out, n.IPAddress)
	return out
}
