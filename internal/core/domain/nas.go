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
	TenantID   TenantID `json:"tenant_id"`  // Isolation Multi-Tenant
	Identifier string   `json:"identifier"` // RADIUS NAS-Identifier
	ShortName  string   `json:"short_name"` // Nom usuel (ex: "Pop-Montreal-01")
	RawIP      net.IP   `json:"ip"`         // Adresse IP source autorisée
	Secret     string   `json:"secret"`     // Shared Secret RADIUS
	Active     bool     `json:"active"`     // État opérationnel

	// Metadata pour le suivi et le verrouillage optimiste
	Version   uint64    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// --- Factory (Constructeur) ---

// NewNAS crée une nouvelle instance de NAS avec les validations métier de base.
func NewNAS(
	id NasID,
	tenantID TenantID,
	identifier string,
	shortName string,
	ip net.IP,
	secret string,
	clock Clock,
) (*NAS, error) {

	if shortName == "" {
		return nil, errors.New("nas short name cannot be empty")
	}
	if ip == nil || ip.IsUnspecified() {
		return nil, errors.New("invalid nas ip address")
	}
	if secret == "" {
		return nil, errors.New("radius secret cannot be empty")
	}

	// Copie défensive de l'IP pour éviter les mutations externes accidentelles
	ipCopy := make(net.IP, len(ip))
	copy(ipCopy, ip)

	now := clock.Now()

	return &NAS{
		ID:         id,
		TenantID:   tenantID,
		Identifier: identifier,
		ShortName:  shortName,
		RawIP:      ipCopy,
		Secret:     secret,
		Active:     true, // Actif par défaut à la création
		Version:    1,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// --- Rehydration (Depuis la DB ou le Cache) ---

// RehydrateNAS reconstruit l'objet sans déclencher les logiques de création (version, date).
func RehydrateNAS(
	id NasID,
	tenantID TenantID,
	identifier string,
	shortName string,
	ip net.IP,
	secret string,
	active bool,
	version uint64,
	createdAt, updatedAt time.Time,
) *NAS {
	return &NAS{
		ID:         id,
		TenantID:   tenantID,
		Identifier: identifier,
		ShortName:  shortName,
		RawIP:      ip,
		Secret:     secret,
		Active:     active,
		Version:    version,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}
}

// --- Logique Métier (Comportements) ---

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

// bumpVersion incrémente la version pour la concurrence et met à jour la date de modification.
func (n *NAS) bumpVersion(clock Clock) {
	n.Version++
	n.UpdatedAt = clock.Now()
}

// --- Getters Spécifiques ---

// IP retourne une copie de l'IP pour protéger l'intégrité de l'agrégat.
func (n *NAS) IP() net.IP {
	if n.RawIP == nil {
		return nil
	}
	out := make(net.IP, len(n.RawIP))
	copy(out, n.RawIP)
	return out
}
