package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Adapter agit comme le pont technique entre Nexora et PostgreSQL.
// Il maintient le pool de connexions ouvert pour toute la durée de vie de l'application.
type Adapter struct {
	// Pool est public pour simplifier l'accès dans les repositories (r.adapter.Pool.Query...)
	Pool *pgxpool.Pool
}

// NewAdapter initialise un pool de connexions robuste optimisé pour la production (Carrier-Grade).
func NewAdapter(ctx context.Context, connString string) (*Adapter, error) {
	// 1. Parsing de la configuration
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("erreur configuration DB: %w", err)
	}

	// 2. Réglages de Performance & Stabilité

	// Limite le nombre de connexions pour protéger la base de données
	config.MaxConns = 25

	// Garde des connexions chaudes pour éviter la latence du handshake SSL/TLS
	config.MinConns = 5

	// Indispensable en Cloud : Force le renouvellement des connexions après 30 min
	// pour éviter les soucis avec les Load Balancers (AWS RDS Proxy, Azure PgBouncer)
	config.MaxConnLifetime = 30 * time.Minute

	// Ferme les connexions inutilisées pour libérer les ressources
	config.MaxConnIdleTime = 15 * time.Minute

	// Vérifie proactivement la santé des connexions inactives
	config.HealthCheckPeriod = 30 * time.Second

	// 3. Création du pool
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("échec création pool: %w", err)
	}

	// 4. Fail Fast : Test de connectivité immédiat avec Timeout
	// Si la DB est down, l'application doit refuser de démarrer immédiatement.
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		return nil, fmt.Errorf("base de données injoignable au démarrage: %w", err)
	}

	return &Adapter{Pool: pool}, nil
}

// Close ferme proprement le pool de connexions (Graceful Shutdown).
func (a *Adapter) Close() {
	if a.Pool != nil {
		a.Pool.Close()
	}
}
