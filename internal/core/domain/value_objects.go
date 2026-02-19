package domain

import (
	"bytes"
	"encoding/json"
	"net"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// Regex de validation
var (
	emailRegex    = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,32}$`)
)

// ------------------ TenantID (UUID) ------------------

type TenantID string

// NewTenantID crée un TenantID (Utilisé par ton code existant)
func NewTenantID(v string) (TenantID, error) {
	if _, err := uuid.Parse(v); err != nil {
		return "", ErrInvalidTenantID
	}
	return TenantID(strings.ToLower(v)), nil
}

// ParseTenantID est un alias vers NewTenantID (Utilisé par le nouveau AuditRepository)
// Cela permet de garder la compatibilité avec tout le projet.
func ParseTenantID(v string) (TenantID, error) {
	return NewTenantID(v)
}

func (t TenantID) String() string             { return string(t) }
func (t TenantID) Equals(other TenantID) bool { return t == other }
func (t TenantID) IsZero() bool {
	return strings.TrimSpace(string(t)) == ""
}

// ------------------ UserID (UUID) ------------------

type UserID string

// ParseUserID valide et convertit une string en UserID
func ParseUserID(v string) (UserID, error) {
	if _, err := uuid.Parse(v); err != nil {
		return "", ErrInvalidUserID
	}
	return UserID(strings.ToLower(v)), nil
}

// NewUserID : Alias pratique si besoin ailleurs
func NewUserID(v string) (UserID, error) {
	return ParseUserID(v)
}

func (u UserID) String() string           { return string(u) }
func (u UserID) Equals(other UserID) bool { return u == other }
func (u UserID) IsZero() bool {
	return strings.TrimSpace(string(u)) == ""
}

// ------------------ Email ------------------

type Email string

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

// ------------------ MAC Address (Struct complexe) ------------------

type MAC struct {
	value []byte
}

// ParseMAC analyse une chaîne (ex: "00:11:22:33:44:55")
func ParseMAC(v string) (MAC, error) {
	mac, err := net.ParseMAC(v)
	if err != nil {
		return MAC{}, ErrInvalidMAC
	}
	return MAC{value: mac}, nil
}

// NewMAC est un alias vers ParseMAC (Pour la cohérence avec le reste du code)
func NewMAC(v string) (MAC, error) {
	return ParseMAC(v)
}

func (m MAC) String() string {
	if m.value == nil {
		return ""
	}
	return net.HardwareAddr(m.value).String()
}

func (m MAC) Equals(other MAC) bool {
	return bytes.Equal(m.value, other.value)
}

// MarshalJSON permet à l'API de renvoyer "00:11:22..." au lieu de base64
func (m MAC) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}

// UnmarshalJSON permet à l'API de lire "00:11:22..." depuis le JSON
func (m *MAC) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := ParseMAC(s)
	if err != nil {
		return err
	}
	m.value = parsed.value
	return nil
}

// ------------------ Autres IDs simples ------------------

type PasswordHash string

func NewPasswordHash(hash string) (PasswordHash, error) {
	if len(hash) < 5 {
		return "", ErrInvalidPasswordHash
	}
	return PasswordHash(hash), nil
}
func (p PasswordHash) String() string { return string(p) }

type PlanID string

func NewPlanID(v string) (PlanID, error) {
	if v == "" {
		return "", ErrInvalidPlanID
	}
	return PlanID(v), nil
}
func (p PlanID) String() string { return string(p) }

type NasID string

func NewNasID(v string) (NasID, error) {
	if v == "" {
		return "", ErrInvalidNasID
	}
	return NasID(v), nil
}
func (n NasID) String() string { return string(n) }

type SessionID string

func NewSessionID(v string) (SessionID, error) {
	if v == "" {
		return "", ErrInvalidSessionID
	}
	return SessionID(v), nil
}
func (s SessionID) String() string { return string(s) }
