package domain

import (
	"bytes"
	"net"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var (
	emailRegex    = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,32}$`)
	uuidRegex     = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[1-5][a-fA-F0-9]{3}-[89abAB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$`)
)

// ------------------ Email ------------------

type Email string // Changement: type de base string

func NewEmail(v string) (Email, error) {
	v = strings.TrimSpace(strings.ToLower(v))
	if !emailRegex.MatchString(v) {
		return "", ErrInvalidEmail
	}
	return Email(v), nil
}

func (e Email) String() string          { return string(e) }
func (e Email) Equals(other Email) bool { return e == other }

// ------------------ Username ------------------

type Username string

func NewUsername(v string) (Username, error) {
	if !usernameRegex.MatchString(v) {
		return "", ErrInvalidUsername
	}
	return Username(v), nil
}

func (u Username) String() string             { return string(u) }
func (u Username) Equals(other Username) bool { return u == other }

// ------------------ MAC (Reste struct car []byte) ------------------

type MAC struct {
	value []byte
}

// NOTE: Pour que le MAC passe en JSON, on doit ajouter MarshalJSON si nécessaire,
// mais Redis stocke souvent le MAC en string. Pour l'instant on garde ta logique struct
// mais on ajoute un Tag JSON sur le champ dans ActiveSession.
func NewMAC(v string) (MAC, error) {
	mac, err := net.ParseMAC(v)
	if err != nil {
		return MAC{}, ErrInvalidMAC
	}
	return MAC{value: mac}, nil
}

func (m MAC) String() string {
	if m.value == nil {
		return ""
	}
	return net.HardwareAddr(m.value).String()
}

// Ajout pour JSON automatique (Optionnel mais utile)
func (m MAC) MarshalJSON() ([]byte, error) {
	return []byte(`"` + m.String() + `"`), nil
}

func (m MAC) Equals(other MAC) bool {
	return bytes.Equal(m.value, other.value)
}

// ------------------ TenantID ------------------

type TenantID string

func NewTenantID(v string) (TenantID, error) {
	// uuid.Parse est robuste : il valide le format mais accepte ton ID de test "1111..."
	if _, err := uuid.Parse(v); err != nil {
		return "", ErrInvalidTenantID
	}
	return TenantID(strings.ToLower(v)), nil
}

func (t TenantID) String() string             { return string(t) }
func (t TenantID) Equals(other TenantID) bool { return t == other }

// ------------------ UserID (Corrigé avec Google UUID) ------------------

type UserID string

func NewUserID(v string) (UserID, error) {
	if _, err := uuid.Parse(v); err != nil {
		return "", ErrInvalidUserID
	}
	return UserID(strings.ToLower(v)), nil
}

func (u UserID) String() string           { return string(u) }
func (u UserID) Equals(other UserID) bool { return u == other }

// ------------------ PasswordHash ------------------

type PasswordHash string

func NewPasswordHash(hash string) (PasswordHash, error) {
	if len(hash) < 32 {
		return "", ErrInvalidPasswordHash
	}
	return PasswordHash(hash), nil
}

func (p PasswordHash) String() string                 { return string(p) }
func (p PasswordHash) Equals(other PasswordHash) bool { return p == other }

// ------------------ PlanID ------------------

type PlanID string

func NewPlanID(v string) (PlanID, error) {
	if v == "" {
		return "", ErrInvalidPlanID
	}
	return PlanID(v), nil
}

func (p PlanID) String() string           { return string(p) }
func (p PlanID) Equals(other PlanID) bool { return p == other }

// ------------------ NasID ------------------

type NasID string

func NewNasID(v string) (NasID, error) {
	if v == "" {
		return "", ErrInvalidNasID
	}
	return NasID(v), nil
}

func (n NasID) String() string          { return string(n) }
func (n NasID) Equals(other NasID) bool { return n == other }

// ------------------ SessionID ------------------

type SessionID string

func NewSessionID(v string) (SessionID, error) {
	if v == "" {
		return "", ErrInvalidSessionID
	}
	return SessionID(v), nil
}

func (s SessionID) String() string              { return string(s) }
func (s SessionID) Equals(other SessionID) bool { return s == other }
