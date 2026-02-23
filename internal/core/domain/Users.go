package domain

import (
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// --- Enums ---
type Role string

const (
	RoleSuperAdmin    Role = "super_admin"
	RoleProviderAdmin Role = "provider_admin"
	RoleResellerAdmin Role = "reseller_admin"
	RoleCustomer      Role = "customer"
)

// --- Aggregate Root: User ---
type User struct {
	id           UserID
	username     Username
	email        Email
	passwordHash PasswordHash
	mac          *MAC
	role         Role
	tenantID     TenantID
	active       bool
	expiredAt    *time.Time
	maxSessions  int
	dataQuota    uint64
	usedData     uint64
	version      uint64
	createdAt    time.Time
	updatedAt    time.Time
	mfaEnabled   bool //
}

// --- Parameter Objects (Anti-S107 SonarQube) ---

type NewUserParams struct {
	ID           UserID
	Username     Username
	Email        Email
	PasswordHash PasswordHash
	Role         Role
	TenantID     TenantID
	MaxSessions  int
	DataQuota    uint64
}

type UserSnapshot struct {
	ID           UserID
	Username     Username
	Email        Email
	PasswordHash PasswordHash
	MAC          *MAC
	Role         Role
	TenantID     TenantID
	Active       bool
	ExpiredAt    *time.Time
	MFAEnabled   bool
	MaxSessions  int
	DataQuota    uint64
	UsedData     uint64
	Version      uint64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// --- Factory ---

// NewUser crée un nouvel utilisateur (utilise NewUserParams pour limiter les arguments).
func NewUser(params NewUserParams, clock Clock) (*User, error) {
	if params.MaxSessions < 0 {
		return nil, errors.New("maxSessions cannot be negative")
	}

	now := clock.Now()
	return &User{
		id:           params.ID,
		username:     params.Username,
		email:        params.Email,
		passwordHash: params.PasswordHash,
		role:         params.Role,
		tenantID:     params.TenantID,
		active:       false,
		maxSessions:  params.MaxSessions,
		dataQuota:    params.DataQuota,
		usedData:     0,
		version:      1,
		createdAt:    now,
		updatedAt:    now,
	}, nil
}

// --- Rehydration ---

// RehydrateUser reconstruit un utilisateur depuis la DB via un Snapshot.
func RehydrateUser(s UserSnapshot) (*User, error) {
	u := &User{
		id:           s.ID,
		username:     s.Username,
		email:        s.Email,
		passwordHash: s.PasswordHash,
		mac:          s.MAC,
		role:         s.Role,
		tenantID:     s.TenantID,
		active:       s.Active,
		expiredAt:    s.ExpiredAt,
		maxSessions:  s.MaxSessions,
		dataQuota:    s.DataQuota,
		usedData:     s.UsedData,
		version:      s.Version,
		createdAt:    s.CreatedAt,
		updatedAt:    s.UpdatedAt,
		mfaEnabled:   s.MFAEnabled,
	}
	if err := u.validateInvariants(); err != nil {
		return nil, err
	}
	return u, nil
}

// --- Business Logic ---

func (u *User) CanAuthenticate(clock Clock) error {
	if !u.active {
		return ErrUserInactive
	}
	if u.IsExpired(clock) {
		return ErrUserExpired
	}
	return nil
}

func (u *User) Activate(clock Clock) error {
	if u.active {
		return nil
	}
	if u.IsExpired(clock) {
		return ErrUserExpired
	}
	u.active = true
	u.bumpVersion(clock)
	return nil
}

func (u *User) Deactivate(clock Clock) {
	if !u.active {
		return
	}
	u.active = false
	u.bumpVersion(clock)
}

func (u *User) Expire(at time.Time, clock Clock) error {
	if at.Before(clock.Now()) {
		return errors.New("expiration date cannot be in the past")
	}
	u.expiredAt = &at
	u.bumpVersion(clock)
	return nil
}

func (u *User) BindMAC(mac MAC, clock Clock) {
	u.mac = &mac
	u.bumpVersion(clock)
}

// ConsumeData augmente la consommation de données.
func (u *User) ConsumeData(bytes uint64, clock Clock) error {
	if bytes == 0 {
		return nil
	}
	if u.usedData+bytes < u.usedData {
		return errors.New("data overflow detected")
	}
	if u.usedData+bytes > u.dataQuota {
		return errors.New("data quota exceeded")
	}
	u.usedData += bytes
	u.bumpVersion(clock)
	return nil
}

func (u *User) IsExpired(clock Clock) bool {
	return u.expiredAt != nil && clock.Now().After(*u.expiredAt)
}

// --- Internal ---
func (u *User) bumpVersion(clock Clock) {
	u.version++
	u.updatedAt = clock.Now()
}

func (u *User) validateInvariants() error {
	if u.maxSessions < 0 {
		return errors.New("negative maxSessions")
	}
	return nil
}

// --- Getters ---
func (u *User) ID() UserID                 { return u.id }
func (u *User) Username() Username         { return u.username }
func (u *User) Email() Email               { return u.email }
func (u *User) PasswordHash() PasswordHash { return u.passwordHash }
func (u *User) MAC() *MAC {
	if u.mac == nil {
		return nil
	}
	return &MAC{value: u.mac.value}
}
func (u *User) Role() Role            { return u.role }
func (u *User) TenantID() TenantID    { return u.tenantID }
func (u *User) IsActive() bool        { return u.active }
func (u *User) ExpiredAt() *time.Time { return u.expiredAt }
func (u *User) DataQuota() uint64     { return u.dataQuota }
func (u *User) UsedData() uint64      { return u.usedData }
func (u *User) MaxSessions() int      { return u.maxSessions }
func (u *User) Version() uint64       { return u.version }
func (u *User) CreatedAt() time.Time  { return u.createdAt }
func (u *User) UpdatedAt() time.Time  { return u.updatedAt }

// (Fonction liée au Value Object PasswordHash)
func (p PasswordHash) Compare(plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(p), []byte(plain))
}

// MFAEnabled indique si l'utilisateur a activé l'authentification multifacteur.
func (u *User) MFAEnabled() bool {
	// Le nom exact dépend de comment tu as nommé le champ privé dans ta struct User
	// Ça peut être u.mfaEnabled, u.MFAEnabled, ou u.snapshot.MFAEnabled
	return u.mfaEnabled
}
