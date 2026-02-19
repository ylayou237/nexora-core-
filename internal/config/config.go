package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	APIPort       string
	DatabaseURL   string
	RedisAddr     string
	RedisPassword string
	JWTSecret     string
	JWTAccessTTL  time.Duration
	RadiusPort    string
	AppEnv        string
}

func Load() *Config {
	// Charge le .env localement
	err := godotenv.Load()
	if err != nil {
		log.Println("ℹ️ Aucun fichier .env trouvé, utilisation des variables système")
	}

	return &Config{
		APIPort:       getEnv("API_PORT", "9000"),
		DatabaseURL:   getEnv("DATABASE_URL", ""),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		JWTSecret:     getEnv("JWT_SECRET", "dev-secret-key-at-least-32-chars"),
		JWTAccessTTL:  parseDuration(getEnv("JWT_ACCESS_TTL", "15m")),
		RadiusPort:    getEnv("RADIUS_AUTH_PORT", "1812"),
		AppEnv:        getEnv("APP_ENV", "development"),
	}
}

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
