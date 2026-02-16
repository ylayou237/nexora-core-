package domain

import (
	"bytes"
	"net"
	"regexp"
	"strings"
)

var (
	emailRegex    = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,32}$`)
	uuidRegex     = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[1-5][a-fA-F0-9]{3}-[89abAB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$`)
)

// ------------------ Email ------------------

type Email struct {
	value string
}

func NewEmail(v string) (Email, error) {
	v = strings.TrimSpace(strings.ToLower(v))
	if !emailRegex.MatchString(v) {
		return Email{}, ErrInvalidEmail
	}
	return Email{value: v}, nil
}

func (e Email) String() string { return e.value }
func (e Email) Equals(other Email) bool {
	return e.value == other.value
}

// ------------------ Username ------------------

type Username struct {
	value string
}

func NewUsername(v string) (Username, error) {
	if !usernameRegex.MatchString(v) {
		return Username{}, ErrInvalidUsername
	}
	return Username{value: v}, nil
}

func (u Username) String() string { return u.value }
func (u Username) Equals(other Username) bool {
	return u.value == other.value
}

// ------------------ MAC ------------------

type MAC struct {
	value []byte
}

func NewMAC(v string) (MAC, error) {
	mac, err := net.ParseMAC(v)
	if err != nil {
		return MAC{}, ErrInvalidMAC
	}
	return MAC{value: append([]byte(nil), mac...)}, nil
}

func (m MAC) Copy() MAC {
	newValue := append([]byte(nil), m.value...)
	return MAC{value: newValue}
}

func (m MAC) Bytes() []byte {
	return append([]byte(nil), m.value...)
}

func (m MAC) String() string {
	return net.HardwareAddr(m.value).String()
}

func (m MAC) Equals(other MAC) bool {
	return bytes.Equal(m.value, other.value)
}

// ------------------ TenantID ------------------

type TenantID struct {
	value string
}

func NewTenantID(v string) (TenantID, error) {
	if !uuidRegex.MatchString(v) {
		return TenantID{}, ErrInvalidTenantID
	}
	return TenantID{value: strings.ToLower(v)}, nil
}

func (t TenantID) String() string             { return t.value }
func (t TenantID) Equals(other TenantID) bool { return t.value == other.value }

// ------------------ UserID ------------------

type UserID struct {
	value string
}

func NewUserID(v string) (UserID, error) {
	if !uuidRegex.MatchString(v) {
		return UserID{}, ErrInvalidUserID
	}
	return UserID{value: strings.ToLower(v)}, nil
}

func (u UserID) String() string           { return u.value }
func (u UserID) Equals(other UserID) bool { return u.value == other.value }

// ------------------ PasswordHash ------------------

type PasswordHash struct {
	value string
}

func NewPasswordHash(hash string) (PasswordHash, error) {
	if len(hash) < 32 {
		return PasswordHash{}, ErrInvalidPasswordHash
	}
	return PasswordHash{value: hash}, nil
}

func (p PasswordHash) String() string                 { return p.value }
func (p PasswordHash) Equals(other PasswordHash) bool { return p.value == other.value }

// ------------------ PlanID ------------------

type PlanID struct {
	value string
}

func NewPlanID(v string) (PlanID, error) {
	if v == "" {
		return PlanID{}, ErrInvalidPlanID
	}
	return PlanID{value: v}, nil
}

func (p PlanID) String() string           { return p.value }
func (p PlanID) Equals(other PlanID) bool { return p.value == other.value }

// ------------------ NasID ------------------

type NasID struct {
	value string
}

func NewNasID(v string) (NasID, error) {
	if v == "" {
		return NasID{}, ErrInvalidNasID
	}
	return NasID{value: v}, nil
}

func (n NasID) String() string          { return n.value }
func (n NasID) Equals(other NasID) bool { return n.value == other.value }

// ------------------ SessionID ------------------

type SessionID struct {
	value string
}

func NewSessionID(v string) (SessionID, error) {
	if v == "" {
		return SessionID{}, ErrInvalidSessionID
	}
	return SessionID{value: v}, nil
}

func (s SessionID) String() string              { return s.value }
func (s SessionID) Equals(other SessionID) bool { return s.value == other.value }
