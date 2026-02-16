package redis

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yvan/nexora-core/internal/core/domain"
)

//go:embed scripts/update_usage.lua
var updateUsageScript string

type SessionRepository struct {
	adapter *Adapter
}

func NewSessionRepository(a *Adapter) *SessionRepository {
	if a == nil {
		panic("Impossible de créer un SessionRepository avec un adapter nil")
	}
	return &SessionRepository{adapter: a}
}

// StartSession initialise une session active dans Redis.
func (r *SessionRepository) StartSession(ctx context.Context, s *domain.ActiveSession) error {
	key := fmt.Sprintf("session:%s", s.ID.String())

	// Protection contre le NIL pointer sur MacAddr
	macStr := ""
	if s.MacAddr != nil {
		macStr = s.MacAddr.String()
	}

	fields := map[string]interface{}{
		"user_id":    s.UserID.String(),
		"nas_ip":     s.NasIP,
		"mac":        macStr, // Utilise la variable sécurisée
		"used_in":    s.InputOctets,
		"used_out":   s.OutputOctets,
		"quota":      s.Policy.DataQuota,
		"expires_at": s.LastAliveAt.Add(s.LeaseDuration).Unix(),
		"started_at": s.StartedAt.Unix(),
	}

	if err := r.adapter.Client.HSet(ctx, key, fields).Err(); err != nil {
		return fmt.Errorf("session_cache: start failed: %w", err)
	}

	return r.adapter.Client.Expire(ctx, key, 24*time.Hour).Err()
}

// GetByID récupère et rehydrate une session.
func (r *SessionRepository) GetByID(ctx context.Context, id domain.SessionID) (*domain.ActiveSession, error) {
	key := fmt.Sprintf("session:%s", id.String())

	data, err := r.adapter.Client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("session_cache: redis error: %w", err)
	}
	if len(data) == 0 {
		return nil, domain.ErrSessionExpired
	}

	// Reconstruction UserID
	uID, err := domain.NewUserID(data["user_id"])
	if err != nil {
		return nil, fmt.Errorf("session_cache: invalid user id")
	}

	// Reconstruction MAC
	macValue, err := domain.NewMAC(data["mac"])
	if err != nil {
		return nil, fmt.Errorf("session_cache: invalid mac address")
	}
	mac := &macValue

	// --- Parsing sécurisé (CORRIGÉ) ---

	usedIn, err := strconv.ParseUint(data["used_in"], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("session_cache: used_in corrupted: %w", err)
	}

	usedOut, err := strconv.ParseUint(data["used_out"], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("session_cache: used_out corrupted: %w", err)
	}

	quota, err := strconv.ParseUint(data["quota"], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("session_cache: quota corrupted: %w", err)
	}

	expiresAtUnix, err := strconv.ParseInt(data["expires_at"], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("session_cache: expires_at corrupted: %w", err)
	}

	startedAtUnix, err := strconv.ParseInt(data["started_at"], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("session_cache: started_at corrupted: %w", err)
	}

	expiresAt := time.Unix(expiresAtUnix, 0)
	startedAt := time.Unix(startedAtUnix, 0)

	remaining := time.Until(expiresAt)
	if remaining <= 0 {
		return nil, domain.ErrSessionExpired
	}

	return domain.RehydrateActiveSession(
		id,
		uID,
		data["nas_ip"],
		mac,
		domain.PolicySnapshot{DataQuota: quota},
		usedIn,
		usedOut,
		0,
		startedAt,
		expiresAt,
		remaining,
	), nil
}

// UpdateUsage exécute le script Lua et rafraîchit le TTL.
func (r *SessionRepository) UpdateUsage(ctx context.Context, id domain.SessionID, delta domain.UsageDelta) error {
	key := fmt.Sprintf("session:%s", id.String())
	now := time.Now().Unix()

	res, err := r.adapter.Client.Eval(
		ctx,
		updateUsageScript,
		[]string{key},
		delta.InputOctets,
		delta.OutputOctets,
		now,
	).Int()

	if err != nil {
		if err == redis.Nil {
			return domain.ErrSessionExpired
		}
		return fmt.Errorf("session_cache: lua execution failed: %w", err)
	}

	switch res {
	case 1:
		if err := r.adapter.Client.Expire(ctx, key, 30*time.Minute).Err(); err != nil {
			return fmt.Errorf("session_cache: ttl refresh failed: %w", err)
		}
		return nil
	case 0:
		return domain.ErrSessionExpired
	case -1:
		return domain.ErrQuotaExceeded
	default:
		return fmt.Errorf("session_cache: unexpected lua signal %d", res)
	}
}

// TerminateSession supprime la session active.
func (r *SessionRepository) TerminateSession(ctx context.Context, id domain.SessionID) error {
	key := fmt.Sprintf("session:%s", id.String())

	if err := r.adapter.Client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("session_cache: terminate failed: %w", err)
	}
	return nil
}

// internal/adapters/secondary/redis/session.go

// Exists vérifie si une session est présente dans Redis
func (r *SessionRepository) Exists(ctx context.Context, id domain.SessionID) (bool, error) {
	// On génère la clé Redis (assure-toi que le format correspond à StartSession)
	key := "session:" + id.String()

	// La méthode Exists de go-redis renvoie le nombre de clés trouvées (0 ou 1)
	count, err := r.adapter.Client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
