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

	// ✅ CORRECTION : Le champ s'appelle maintenant IPAddress et est public
	IPAddress net.IP `json:"ip_address"`

	Secret string `json:"secret"`
	Active bool   `json:"active"`

	// Metadata
	Version   uint64    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// --- Factory (Constructeur) ---

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

	// Copie défensive
	ipCopy := make(net.IP, len(ip))
	copy(ipCopy, ip)

	now := clock.Now()

	return &NAS{
		ID:         id,
		TenantID:   tenantID,
		Identifier: identifier,
		ShortName:  shortName,
		// ✅ Assignation au nouveau champ
		IPAddress: ipCopy,
		Secret:    secret,
		Active:    true,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// --- Rehydration ---

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
		// ✅ Assignation au nouveau champ
		IPAddress: ip,
		Secret:    secret,
		Active:    active,
		Version:   version,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
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
// (Garde cette méthode si tu veux un accesseur sécurisé, mais Redis utilisera le champ direct)
func (n *NAS) IP() net.IP {
	// ✅ Utilisation du nouveau champ
	if n.IPAddress == nil {
		return nil
	}
	out := make(net.IP, len(n.IPAddress))
	copy(out, n.IPAddress)
	return out
}
