package postgres

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yvan/nexora-core/internal/core/domain"
)

// userModel est le DTO qui reflète EXACTEMENT la table "users" de PostgreSQL.
// Les tags `db` sont utilisés par pgx pour le mapping automatique.
type userModel struct {
	ID           string     `db:"id"`
	TenantID     string     `db:"tenant_id"`
	Username     string     `db:"username"`
	Email        *string    `db:"email"` // <-- pointeur pour gérer NULL
	PasswordHash string     `db:"password_hash"`
	Role         string     `db:"role"`
	MAC          *string    `db:"mac_address"`
	Active       bool       `db:"active"`
	ExpiredAt    *time.Time `db:"expired_at"`
	MaxSessions  int        `db:"max_sessions"`
	DataQuota    int64      `db:"data_quota"`
	UsedData     int64      `db:"used_data"`
	Version      int64      `db:"version"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
}

// UserRepository implémente l'interface ports.UserRepository.
type UserRepository struct {
	adapter *Adapter
}

// NewUserRepository constructeur.
func NewUserRepository(a *Adapter) *UserRepository {
	return &UserRepository{adapter: a}
}

func (r *UserRepository) GetByUsername(ctx context.Context, tID domain.TenantID, u domain.Username) (*domain.User, error) {
	query := `
    SELECT 
        id, tenant_id, username, email, 
        password_hash, 
        role, mac_address, active, expired_at, 
        max_sessions, data_quota, used_data, 
        version, created_at, updated_at
    FROM users 
    WHERE tenant_id = $1 AND username = $2
    LIMIT 1`

	// ⚠️ Utilise QueryRow pour éviter les problèmes de rows.Close()
	row := r.adapter.Pool.QueryRow(ctx, query, tID.String(), u.String())

	var model userModel
	err := row.Scan(
		&model.ID,
		&model.TenantID,
		&model.Username,
		&model.Email,
		&model.PasswordHash,
		&model.Role,
		&model.MAC,
		&model.Active,
		&model.ExpiredAt,
		&model.MaxSessions,
		&model.DataQuota,
		&model.UsedData,
		&model.Version,
		&model.CreatedAt,
		&model.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("🔍 DEBUG: Utilisateur non trouvé en DB pour %s (tenant %s)", u.String(), tID.String())
			return nil, domain.ErrUserNotFound
		}
		log.Printf("❌ DEBUG: Erreur DB pour %s : %v", u.String(), err)
		return nil, err
	}

	log.Printf("🔍 DEBUG DB: Utilisateur trouvé -> %s | Hash = [%s]", model.Username, model.PasswordHash)

	return r.mapToDomain(&model)
}

// Create insère un nouvel utilisateur.
func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	query := `
		INSERT INTO users (
			id, tenant_id, username, email, password_hash, role, 
			mac_address, active, expired_at, max_sessions, 
			data_quota, used_data, version, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`

	// Gestion du pointeur pour le MAC address (peut être NULL)
	var mac *string
	if u.MAC() != nil {
		s := u.MAC().String()
		mac = &s
	}

	_, err := r.adapter.Pool.Exec(ctx, query,
		u.ID().String(),
		u.TenantID().String(),
		u.Username().String(),
		u.Email().String(),
		u.PasswordHash().String(),
		string(u.Role()),
		mac,
		u.IsActive(),
		u.ExpiredAt(),
		u.MaxSessions(),
		int64(u.DataQuota()), // Cast uint64 -> int64 (Postgres BIGINT est signé)
		int64(0),             // UsedData commence toujours à 0 à la création
		u.Version(),
		u.CreatedAt(),
		u.UpdatedAt(),
	)

	if err != nil {
		var pgErr *pgconn.PgError
		// Code 23505 = Unique Violation (ex: username déjà pris)
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return errors.New("conflict: user already exists in this tenant")
		}
		return err
	}

	return nil
}

// Update met à jour un utilisateur avec Optimistic Locking.
func (r *UserRepository) Update(ctx context.Context, u *domain.User) error {
	query := `
		UPDATE users SET 
			email = $1, password_hash = $2, role = $3, 
			mac_address = $4, active = $5, expired_at = $6, 
			max_sessions = $7, data_quota = $8, used_data = $9, 
			version = version + 1, updated_at = NOW()
		WHERE id = $10 AND version = $11`

	var mac *string
	if u.MAC() != nil {
		s := u.MAC().String()
		mac = &s
	}

	// On persiste la consommation actuelle (cast uint64 -> int64)
	usedData := int64(u.UsedData())

	tag, err := r.adapter.Pool.Exec(ctx, query,
		u.Email().String(),
		u.PasswordHash().String(),
		string(u.Role()),
		mac,
		u.IsActive(),
		u.ExpiredAt(),
		u.MaxSessions(),
		int64(u.DataQuota()),
		usedData,
		u.ID().String(),
		u.Version(),
	)

	if err != nil {
		return err
	}

	// Si aucune ligne n'est touchée, c'est que la version a changé (concurrence)
	if tag.RowsAffected() == 0 {
		return errors.New("optimistic locking failure: user modified concurrently")
	}

	return nil
}

// Delete supprime un utilisateur.
func (r *UserRepository) Delete(ctx context.Context, id domain.UserID) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.adapter.Pool.Exec(ctx, query, id.String())
	return err
}

// --- MAPPING (Anti-Corruption Layer) ---

// --- MAPPING (Anti-Corruption Layer) ---

func (r *UserRepository) mapToDomain(m *userModel) (*domain.User, error) {
	// ID
	id, err := domain.NewUserID(m.ID)
	if err != nil {
		return nil, err
	}

	// TenantID
	tID, err := domain.NewTenantID(m.TenantID)
	if err != nil {
		return nil, err
	}

	// Username
	username, err := domain.NewUsername(m.Username)
	if err != nil {
		return nil, err
	}

	// Email optionnel
	var email domain.Email
	if m.Email != nil {
		email, err = domain.NewEmail(*m.Email)
		if err != nil {
			return nil, err
		}
	} else {
		email = domain.Email("") // email vide si NULL
	}

	// Password hash
	hash, err := domain.NewPasswordHash(m.PasswordHash)
	if err != nil {
		return nil, err
	}

	// MAC optionnelle
	var mac *domain.MAC
	if m.MAC != nil {
		val, err := domain.NewMAC(*m.MAC)
		if err != nil {
			return nil, err
		}
		mac = &val
	}

	// 🔹 Debug
	log.Printf("🔍 [DEBUG] Hash depuis DB: [%s]", m.PasswordHash)
	log.Printf("🔍 [DEBUG] Hash dans le Domaine: [%s]", hash.String())

	// Rehydrate avec les BONNES variables parsées et castées
	return domain.RehydrateUser(domain.UserSnapshot{
		ID:           id,
		Username:     username,
		Email:        email,
		PasswordHash: hash, // la variable hash parsée au dessus
		MAC:          mac,
		Role:         domain.Role(m.Role), // cast string -> domain.Role
		TenantID:     tID,                 // la variable tID parsée au dessus
		Active:       m.Active,
		ExpiredAt:    m.ExpiredAt,
		MaxSessions:  m.MaxSessions,
		DataQuota:    uint64(m.DataQuota), // cast int64 de la DB vers uint64
		UsedData:     uint64(m.UsedData),  // cast int64 de la DB vers uint64
		Version:      uint64(m.Version),   // cast int64 de la DB vers uint64
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	})
} // 👈 L'accolade manquante était ici !

// GetByID récupère un utilisateur par son ID unique.
func (r *UserRepository) GetByID(ctx context.Context, id domain.UserID) (*domain.User, error) {
	query := `
		SELECT id, tenant_id, username, email, password_hash, role, 
			   mac_address, active, expired_at, max_sessions, 
			   data_quota, used_data, version, created_at, updated_at
		FROM users 
		WHERE id = $1 
		LIMIT 1`

	rows, _ := r.adapter.Pool.Query(ctx, query, id.String())
	model, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[userModel])

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	return r.mapToDomain(model)
}
