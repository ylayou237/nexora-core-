package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// 1. Chargement de l'environnement
	_ = godotenv.Load()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("❌ DATABASE_URL est absente du fichier .env")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		log.Fatalf("❌ Impossible de se connecter à Neon: %v", err)
	}
	defer conn.Close(ctx)

	// IDs fixes pour garantir la reproductibilité des tests
	const (
		testTenantID = "550e8400-e29b-41d4-a716-446655440000"
		testUserID   = "660f8400-e29b-41d4-a716-446655441111"
		username     = "yvan_clean"
		password     = "password123"
	)

	fmt.Println("🌱 Début du peuplement de la base de données Nexora...")

	// 2. Insertion/Update du Tenant
	tenantQuery := `
		INSERT INTO tenants (id, name, type, status, portal_enabled, created_at, updated_at) 
		VALUES ($1, 'Nexora Global', 'provider', 'active', true, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET 
			name = EXCLUDED.name, 
			status = 'active',
			updated_at = NOW();
	`
	_, err = conn.Exec(ctx, tenantQuery, testTenantID)
	if err != nil {
		log.Fatalf("❌ Erreur lors de la création du Tenant : %v", err)
	}
	fmt.Println("✅ Tenant 'Nexora Global' prêt.")

	// 3. Préparation sécurisée du mot de passe
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("❌ Erreur lors du hashage du mot de passe : %v", err)
	}

	// 4. Insertion/Update de l'Utilisateur Admin
	// Inclut les limites Radius et la gestion de version pour ton repository
	userQuery := `
		INSERT INTO users (
			id, tenant_id, username, email, password_hash, 
			role, active, max_sessions, data_quota, used_data, 
			version, created_at, updated_at
		) 
		VALUES ($1, $2, $3, $4, $5, 'admin', true, 5, 10737418240, 0, 1, NOW(), NOW())
		ON CONFLICT (tenant_id, username) DO UPDATE SET 
			password_hash = EXCLUDED.password_hash,
			active = true,
			data_quota = EXCLUDED.data_quota,
			updated_at = NOW(),
			version = users.version + 1;
	`
	// 10737418240 octets = 10 GB
	_, err = conn.Exec(ctx, userQuery,
		testUserID,
		testTenantID,
		username,
		"admin@nexora.io",
		string(hash),
	)
	if err != nil {
		log.Fatalf("❌ Erreur lors de la création de l'Utilisateur : %v", err)
	}

	fmt.Println("--------------------------------------------------")
	fmt.Println("🚀 DATABASE SEEDED & CARRIER-GRADE READY")
	fmt.Printf("🏢 Tenant ID : %s\n", testTenantID)
	fmt.Printf("👤 Username  : %s\n", username)
	fmt.Printf("🔑 Password  : %s\n", password)
	fmt.Printf("📊 Quota     : 10 GB\n")
	fmt.Printf("🛡️  Role      : Admin\n")
	fmt.Println("--------------------------------------------------")
}
