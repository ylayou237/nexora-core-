package redis

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"
	"time"

	"github.com/yvan/nexora-core/internal/core/domain"
)

//go:embed scripts/update_usage.lua
var updateUsageScript string

type SessionRepository struct {
	adapter *Adapter
}

func NewSessionRepository(a *Adapter) *SessionRepository {
	return &SessionRepository{adapter: a}
}

// StartSession initialise la session avec des clés alignées sur le script Lua.
func (r *SessionRepository) StartSession(ctx context.Context, s *domain.ActiveSession) error {
	key := fmt.Sprintf("session:%s", s.ID.String())

	// Mapping STRICT pour le script Lua : used_in, used_out, quota, expires_at
	fields := map[string]interface{}{
		"user_id":    s.UserID.String(),
		"nas_ip":     s.NasIP,
		"mac":        s.MacAddr.String(),
		"used_in":    s.InputOctets,
		"used_out":   s.OutputOctets,
		"quota":      s.Policy.DataQuota,
		"expires_at": s.LastAliveAt.Add(s.LeaseDuration).Unix(),
		"started_at": s.StartedAt.Unix(),
	}

	// Stockage atomique du Hash
	if err := r.adapter.Client.HSet(ctx, key, fields).Err(); err != nil {
		return fmt.Errorf("session_cache: failed to start session: %w", err)
	}

	// Sécurité RAM : TTL physique de 24h (la session sera rafraîchie par les Heartbeats)
	return r.adapter.Client.Expire(ctx, key, 24*time.Hour).Err()
}

// GetByID récupère et réhydrate une session depuis Redis.
func (r *SessionRepository) GetByID(ctx context.Context, sessionID string) (*domain.ActiveSession, error) {
	key := fmt.Sprintf("session:%s", sessionID)

	data, err := r.adapter.Client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("session_cache: redis error: %w", err)
	}

	// Si la map est vide, la session n'existe pas ou a expiré
	if len(data) == 0 {
		return nil, domain.ErrSessionExpired
	}

	// 1. Reconstitution sécurisée des Value Objects (Validation Domain)
	sID, errSID := domain.NewSessionID(sessionID)
	uID, errUID := domain.NewUserID(data["user_id"])
	if errSID != nil || errUID != nil {
		return nil, fmt.Errorf("session_cache: ID corruption detected")
	}

	// 2. Parsing des valeurs numériques (Redis stocke du string)
	usedIn, _ := strconv.ParseUint(data["used_in"], 10, 64)
	usedOut, _ := strconv.ParseUint(data["used_out"], 10, 64)
	quota, _ := strconv.ParseUint(data["quota"], 10, 64)
	expiresAt, _ := strconv.ParseInt(data["expires_at"], 10, 64)
	startedAt, _ := strconv.ParseInt(data["started_at"], 10, 64)

	// 3. Rehydration de l'agrégat
	return domain.RehydrateActiveSession(
		sID,
		uID,
		data["nas_ip"],
		nil, // Le MAC peut être ajouté ici via domain.NewMAC si besoin
		domain.PolicySnapshot{DataQuota: quota},
		usedIn,
		usedOut,
		0, // SessionTime
		time.Unix(startedAt, 0),
		time.Unix(expiresAt, 0),
		time.Duration(expiresAt-time.Now().Unix())*time.Second,
	), nil
}

// UpdateUsage exécute le script Lua et rafraîchit le bail de vie (TTL).
func (r *SessionRepository) UpdateUsage(ctx context.Context, sessionID string, delta domain.UsageDelta) error {
	key := fmt.Sprintf("session:%s", sessionID)
	now := time.Now().Unix()

	// Exécution atomique
	res, err := r.adapter.Client.Eval(ctx, updateUsageScript, []string{key},
		delta.InputOctets,
		delta.OutputOctets,
		now,
	).Int()

	if err != nil {
		return fmt.Errorf("session_cache: lua execution failed: %w", err)
	}

	// Gestion des signaux de retour Lua
	switch res {
	case 1:
		// Succès : On repousse le TTL (Sliding Window)
		r.adapter.Client.Expire(ctx, key, 30*time.Minute)
		return nil
	case 0:
		return domain.ErrSessionExpired
	case -1:
		return domain.ErrQuotaExceeded
	default:
		return fmt.Errorf("session_cache: unexpected signal %d", res)
	}
}

// TerminateSession supprime la clé de la mémoire vive.
func (r *SessionRepository) TerminateSession(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return r.adapter.Client.Del(ctx, key).Err()
}
