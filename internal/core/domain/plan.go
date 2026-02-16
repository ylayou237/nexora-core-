package domain

import (
	"errors"
	"time"
)

// Note: PlanID est défini dans value_objects.go.
// Ne pas le redéfinir ici pour éviter "redeclared in this block".

// --- Aggregate Root: Plan ---

type Plan struct {
	id          PlanID
	tenantID    TenantID // 👈 Isolation Multi-Tenant confirmée
	name        string
	description string

	// Policy Limits (Configuration)
	dataQuota   uint64 // bytes
	maxUpload   uint64 // bps
	maxDownload uint64 // bps
	maxSessions int    // Coherent avec UserIdentity
	timeQuota   time.Duration

	// Metadata
	version   uint64
	createdAt time.Time
	updatedAt time.Time
}

// --- Factory ---

func NewPlan(
	id PlanID,
	tenantID TenantID,
	name string,
	description string,
	dataQuota uint64,
	maxUpload uint64,
	maxDownload uint64,
	maxSessions int,
	timeQuota time.Duration,
	clock Clock,
) (*Plan, error) {

	if name == "" {
		return nil, errors.New("plan name cannot be empty")
	}
	if maxSessions < 0 {
		return nil, errors.New("maxSessions cannot be negative")
	}

	now := clock.Now()

	return &Plan{
		id:          id,
		tenantID:    tenantID,
		name:        name,
		description: description,
		dataQuota:   dataQuota,
		maxUpload:   maxUpload,
		maxDownload: maxDownload,
		maxSessions: maxSessions,
		timeQuota:   timeQuota,
		version:     1,
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

// --- Rehydration ---

func RehydratePlan(
	id PlanID,
	tenantID TenantID,
	name, description string,
	dataQuota, maxUpload, maxDownload uint64,
	maxSessions int,
	timeQuota time.Duration,
	version uint64,
	createdAt, updatedAt time.Time,
) (*Plan, error) {

	p := &Plan{
		id:          id,
		tenantID:    tenantID,
		name:        name,
		description: description,
		dataQuota:   dataQuota,
		maxUpload:   maxUpload,
		maxDownload: maxDownload,
		maxSessions: maxSessions,
		timeQuota:   timeQuota,
		version:     version,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}

	if err := p.validateInvariants(); err != nil {
		return nil, err
	}

	return p, nil
}

// --- Business Logic (Mutations) ---

// UpdateDetails permet de changer le nom/description (Admin)
func (p *Plan) UpdateDetails(name, description string, clock Clock) error {
	if name == "" {
		return errors.New("plan name cannot be empty")
	}
	p.name = name
	p.description = description
	p.bumpVersion(clock)
	return nil
}

// UpdateLimits permet de changer les quotas (Admin)
func (p *Plan) UpdateLimits(
	dataQuota, maxUpload, maxDownload uint64,
	maxSessions int,
	timeQuota time.Duration,
	clock Clock,
) error {
	if maxSessions < 0 {
		return errors.New("maxSessions cannot be negative")
	}

	p.dataQuota = dataQuota
	p.maxUpload = maxUpload
	p.maxDownload = maxDownload
	p.maxSessions = maxSessions
	p.timeQuota = timeQuota
	p.bumpVersion(clock)
	return nil
}

// --- Internal ---

func (p *Plan) validateInvariants() error {
	if p.maxSessions < 0 {
		return errors.New("invariant violation: negative maxSessions")
	}
	return nil
}

func (p *Plan) bumpVersion(clock Clock) {
	p.version++
	p.updatedAt = clock.Now()
}

// --- Getters ---

func (p *Plan) ID() PlanID               { return p.id }
func (p *Plan) TenantID() TenantID       { return p.tenantID }
func (p *Plan) Name() string             { return p.name }
func (p *Plan) Description() string      { return p.description }
func (p *Plan) DataQuota() uint64        { return p.dataQuota }
func (p *Plan) MaxUpload() uint64        { return p.maxUpload }
func (p *Plan) MaxDownload() uint64      { return p.maxDownload }
func (p *Plan) MaxSessions() int         { return p.maxSessions }
func (p *Plan) TimeQuota() time.Duration { return p.timeQuota }
func (p *Plan) Version() uint64          { return p.version }
func (p *Plan) CreatedAt() time.Time     { return p.createdAt }
func (p *Plan) UpdatedAt() time.Time     { return p.updatedAt }
