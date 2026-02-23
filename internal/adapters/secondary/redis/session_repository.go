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

const sessionKeyPattern = "session:%s"

type SessionRepository struct {
	// UniversalClient permet une flexibilité totale (Simple, Cluster, Failover)
	client redis.UniversalClient
}

func NewSessionRepository(client redis.UniversalClient) *SessionRepository {
	if client == nil {
		panic("Impossible de créer un SessionRepository avec un client redis nil")
	}
	return &SessionRepository{client: client}
}

// StartSession enregistre une session active avec un pipeline atomique
func (r *SessionRepository) StartSession(ctx context.Context, s *domain.ActiveSession) error {
	key := fmt.Sprintf(sessionKeyPattern, s.ID.String())

	macStr := ""
	if s.MacAddr != nil {
		macStr = s.MacAddr.String()
	}

	fields := map[string]interface{}{
		"user_id":    s.UserID.String(),
		"nas_ip":     s.NasIP,
		"mac":        macStr,
		"used_in":    s.InputOctets,
		"used_out":   s.OutputOctets,
		"quota":      s.Policy.DataQuota,
		"expires_at": s.LastAliveAt.Add(s.LeaseDuration).Unix(),
		"started_at": s.StartedAt.Unix(),
	}

	// Pipeline : réduit la latence réseau en groupant HSET et EXPIRE
	pipe := r.client.Pipeline()
	pipe.HSet(ctx, key, fields)
	pipe.Expire(ctx, key, 24*time.Hour)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("session_repository: start pipeline failed: %w", err)
	}

	return nil
}

// GetByID hydrate une session depuis Redis avec gestion d'expiration
func (r *SessionRepository) GetByID(ctx context.Context, id domain.SessionID) (*domain.ActiveSession, error) {
	key := fmt.Sprintf(sessionKeyPattern, id.String())

	data, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("session_repository: redis error: %w", err)
	}
	if len(data) == 0 {
		return nil, domain.ErrSessionExpired
	}

	uID, err := domain.NewUserID(data["user_id"])
	if err != nil {
		return nil, fmt.Errorf("session_repository: corrupted user_id")
	}

	var mac *domain.MAC
	if data["mac"] != "" {
		macValue, err := domain.NewMAC(data["mac"])
		if err == nil {
			mac = &macValue
		}
	}

	// Parsing sécurisé des données scalaires
	usedIn, _ := strconv.ParseUint(data["used_in"], 10, 64)
	usedOut, _ := strconv.ParseUint(data["used_out"], 10, 64)
	quota, _ := strconv.ParseUint(data["quota"], 10, 64)
	expiresAtUnix, _ := strconv.ParseInt(data["expires_at"], 10, 64)
	startedAtUnix, _ := strconv.ParseInt(data["started_at"], 10, 64)

	expiresAt := time.Unix(expiresAtUnix, 0)
	remaining := time.Until(expiresAt)

	if remaining <= 0 {
		return nil, domain.ErrSessionExpired
	}

	snapshot := domain.SessionSnapshot{
		ID:            id,
		UserID:        uID,
		NasIP:         data["nas_ip"],
		MacAddr:       mac,
		Policy:        domain.PolicySnapshot{DataQuota: quota},
		InputOctets:   usedIn,
		OutputOctets:  usedOut,
		StartedAt:     time.Unix(startedAtUnix, 0),
		LastAliveAt:   expiresAt,
		LeaseDuration: remaining,
	}

	return domain.RehydrateActiveSession(snapshot), nil
}

// UpdateUsage utilise un script LUA pour garantir l'atomicité de la mise à jour des quotas
func (r *SessionRepository) UpdateUsage(ctx context.Context, id domain.SessionID, delta domain.UsageDelta) error {
	key := fmt.Sprintf(sessionKeyPattern, id.String())
	now := time.Now().Unix()

	res, err := r.client.Eval(
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
		return fmt.Errorf("session_repository: lua execution failed: %w", err)
	}

	// Interprétation des signaux de retour du script LUA
	switch res {
	case 1:
		// Succès : rafraîchissement glissant du TTL Redis (30 min)
		_ = r.client.Expire(ctx, key, 30*time.Minute).Err()
		return nil
	case 0:
		return domain.ErrSessionExpired
	case -1:
		return domain.ErrQuotaExceeded
	default:
		return fmt.Errorf("session_repository: unexpected signal %d", res)
	}
}

func (r *SessionRepository) TerminateSession(ctx context.Context, id domain.SessionID) error {
	key := fmt.Sprintf(sessionKeyPattern, id.String())
	return r.client.Del(ctx, key).Err()
}

func (r *SessionRepository) Exists(ctx context.Context, id domain.SessionID) (bool, error) {
	key := fmt.Sprintf(sessionKeyPattern, id.String())
	count, err := r.client.Exists(ctx, key).Result()
	return count > 0, err
}

// Health implémente le check de santé pour AuthService.CheckIntegrity
func (r *SessionRepository) Health(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}
