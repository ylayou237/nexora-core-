package domain

import (
	"errors"
	"time"
)

// --- Aggregate Root: Plan ---

type Plan struct {
	id          PlanID
	tenantID    TenantID
	name        string
	description string

	// Policy Limits
	dataQuota   uint64
	maxUpload   uint64
	maxDownload uint64
	maxSessions int
	timeQuota   time.Duration

	// Metadata
	version   uint64
	createdAt time.Time
	updatedAt time.Time
}

// --- Parameter Objects (Anti-S107) ---

// NewPlanParams contient les données nécessaires pour créer un forfait.
type NewPlanParams struct {
	ID          PlanID
	TenantID    TenantID
	Name        string
	Description string
	DataQuota   uint64
	MaxUpload   uint64
	MaxDownload uint64
	MaxSessions int
	TimeQuota   time.Duration
}

// PlanSnapshot contient l'état complet du forfait pour la réhydratation (DB -> Domain).
type PlanSnapshot struct {
	ID          PlanID
	TenantID    TenantID
	Name        string
	Description string
	DataQuota   uint64
	MaxUpload   uint64
	MaxDownload uint64
	MaxSessions int
	TimeQuota   time.Duration
	Version     uint64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// --- Factory ---

// NewPlan utilise maintenant une struct de paramètres.
func NewPlan(params NewPlanParams, clock Clock) (*Plan, error) {
	if params.Name == "" {
		return nil, errors.New("plan name cannot be empty")
	}
	if params.MaxSessions < 0 {
		return nil, errors.New("maxSessions cannot be negative")
	}

	now := clock.Now()

	return &Plan{
		id:          params.ID,
		tenantID:    params.TenantID,
		name:        params.Name,
		description: params.Description,
		dataQuota:   params.DataQuota,
		maxUpload:   params.MaxUpload,
		maxDownload: params.MaxDownload,
		maxSessions: params.MaxSessions,
		timeQuota:   params.TimeQuota,
		version:     1,
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

// --- Rehydration ---

// RehydratePlan utilise un Snapshot pour passer d'un seul coup toutes les données de la DB.
func RehydratePlan(s PlanSnapshot) (*Plan, error) {
	p := &Plan{
		id:          s.ID,
		tenantID:    s.TenantID,
		name:        s.Name,
		description: s.Description,
		dataQuota:   s.DataQuota,
		maxUpload:   s.MaxUpload,
		maxDownload: s.MaxDownload,
		maxSessions: s.MaxSessions,
		timeQuota:   s.TimeQuota,
		version:     s.Version,
		createdAt:   s.CreatedAt,
		updatedAt:   s.UpdatedAt,
	}

	if err := p.validateInvariants(); err != nil {
		return nil, err
	}

	return p, nil
}

// --- Business Logic (Mutations) ---

func (p *Plan) UpdateDetails(name, description string, clock Clock) error {
	if name == "" {
		return errors.New("plan name cannot be empty")
	}
	p.name = name
	p.description = description
	p.bumpVersion(clock)
	return nil
}

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
