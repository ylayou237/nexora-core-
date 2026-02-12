package domain

import "time"

type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusExpired   UserStatus = "expired"
	UserStatusDisabled  UserStatus = "disabled"
)

type User struct {
	ID              UserID       `json:"id"`
	TenantID        TenantID     `json:"tenant_id"`
	PlanID          *PlanID      `json:"plan_id"` // Pointeur car nullable
	Username        Username     `json:"username"`
	PasswordHash    string       `json:"-"` // Ne jamais exposer
	ClearTextPass   string       `json:"-"` // Pour PAP/CHAP
	FirstName       string       `json:"first_name"`
	LastName        string       `json:"last_name"`
	Email           Email        `json:"email"`
	Phone           string       `json:"phone"`
	Address         string       `json:"address"`
	Balance         float64      `json:"balance"`
	Status          UserStatus   `json:"status"`
	MaxSessions     int          `json:"max_sessions"`
	SimultaneousUse bool         `json:"simultaneous_use"`
	AllowedMACs     []MACAddress `json:"allowed_macs"`

	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}

func (u *User) CanLogin() bool {
	return u.Status == UserStatusActive
}
