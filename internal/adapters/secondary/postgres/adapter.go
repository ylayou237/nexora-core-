package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Adapter agit comme le pont technique entre notre application et PostgreSQL Cloud.
// Il implémente le pattern Singleton pour le pool de connexions et gère le cycle de vie.
type Adapter struct {
	pool *pgxpool.Pool
}

// NewAdapter initialise un pool de connexions robuste optimisé pour le Cloud.
// Le paramètre connString doit être au format : postgres://user:password@host:port/dbname
func NewAdapter(ctx context.Context, connString string) (*Adapter, error) {
	// 1. Analyse et préparation de la configuration
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de l'analyse de la config : %w", err)
	}

	// 2. Réglages Carrier-Grade pour la performance et la stabilité
	// MaxConns : Limite le nombre de connexions pour ne pas saturer l'instance Cloud.
	config.MaxConns = 25
	// MinConns : Maintient des connexions pré-établies pour éliminer la latence du handshake TLS/SSL.
	config.MinConns = 5
	// HealthCheckPeriod : Vérifie périodiquement la santé des connexions inactives.
	config.HealthCheckPeriod = 30 * time.Second
	// MaxConnIdleTime : Ferme les connexions inutilisées pour libérer les ressources du Cloud.
	config.MaxConnIdleTime = 15 * time.Minute

	// 3. Création du pool
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("impossible de créer le pool de connexions : %w", err)
	}

	// 4. Test de connectivité immédiat (Fail Fast)
	// On utilise un timeout court pour ne pas bloquer le démarrage du service.
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		return nil, fmt.Errorf("la base de données Cloud est injoignable : %w", err)
	}

	return &Adapter{pool: pool}, nil
}

// Close assure une fermeture propre des connexions (Graceful Shutdown).
func (a *Adapter) Close() {
	if a.pool != nil {
		a.pool.Close()
	}
}

// GetPool expose l'accès direct au pool pour les repositories.
func (a *Adapter) GetPool() *pgxpool.Pool {
	return a.pool
}