package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// API & Env
	AppEnv  string
	APIPort string

	// Bases de données
	DatabaseURL   string
	RedisAddr     string
	RedisPassword string

	// Sécurité JWT (RS256)
	JWTIssuer      string
	JWTAccessTTL   time.Duration
	JWTRefreshTTL  time.Duration
	PrivateKeyPath string
	PublicKeyPath  string

	// Worker Pool Audit (SIEM)
	AuditWorkerCount      int
	AuditQueueSize        int
	AuthMaxFailedAttempts int
	AuthLockDuration      time.Duration
	AuthWindowDuration    time.Duration
	// Radius
	RadiusPort string
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		AppEnv:  getEnv("APP_ENV", "development"),
		APIPort: getEnv("API_PORT", "9000"),

		DatabaseURL:   getEnv("DATABASE_URL", ""),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),

		// Synchronisé avec ton .env
		JWTIssuer:     getEnv("JWT_ISSUER", "nexora-core"),
		JWTAccessTTL:  parseDuration(getEnv("JWT_ACCESS_TTL", "15m")),
		JWTRefreshTTL: parseDuration(getEnv("JWT_REFRESH_TTL", "168h")),

		// Chemins vers tes fichiers .pem créés précédemment
		PrivateKeyPath: "./certs/private.pem",
		PublicKeyPath:  "./certs/public.pem",

		// Configuration du Worker Pool
		AuditWorkerCount: parseUint(getEnv("AUDIT_WORKER_COUNT", "5")),
		AuditQueueSize:   parseUint(getEnv("AUDIT_QUEUE_SIZE", "100")),

		RadiusPort: getEnv("RADIUS_AUTH_PORT", "1812"),
		// Anti-bruteforce / Lockout
		AuthMaxFailedAttempts: parseUint(getEnv("AUTH_MAX_FAILED_ATTEMPTS", "5")),
		AuthLockDuration:      parseDuration(getEnv("AUTH_LOCK_DURATION", "15m")),
		AuthWindowDuration:    parseDuration(getEnv("AUTH_WINDOW_DURATION", "1h")),
	}
}

// Helpers
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 15 * time.Minute
	}
	return d
}

func parseUint(s string) int {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil {
		return 5
	}
	return n
}
