package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// 1. Chargement de l'environnement
	_ = godotenv.Load()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL est absente du fichier .env")
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		log.Fatalf("Impossible de se connecter à Neon: %v", err)
	}
	defer conn.Close(ctx)

	// IDs fixes pour tes futurs tests API/RADIUS
	const (
		testTenantID = "550e8400-e29b-41d4-a716-446655440000"
		testUserID   = "660f8400-e29b-41d4-a716-446655441111"
	)

	fmt.Println("🌱 Début du peuplement de la base de données...")

	// 2. Insertion du Tenant (Adapté au nouveau schéma)
	// On respecte les contraintes : type IN ('provider', 'reseller', 'customer')
	tenantQuery := `
		INSERT INTO tenants (id, name, type, status, portal_enabled, created_at, updated_at) 
		VALUES ($1, 'Nexora Global', 'provider', 'active', true, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, status = 'active';
	`
	_, err = conn.Exec(ctx, tenantQuery, testTenantID)
	if err != nil {
		log.Fatalf("❌ Erreur lors de la création du Tenant : %v", err)
	}

	// 3. Préparation du mot de passe
	password := "password123"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	// 4. Insertion de l'Utilisateur (Adapté au nouveau schéma)
	// On remplit les champs Radius : data_quota, max_sessions, etc.
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
			data_quota = EXCLUDED.data_quota;
	`
	// 10737418240 octets = 10 GB de quota
	_, err = conn.Exec(ctx, userQuery,
		testUserID,
		testTenantID,
		"yvan_clean",
		"yvan@nexora.io",
		string(hash),
	)
	if err != nil {
		log.Fatalf("❌ Erreur lors de la création de l'Utilisateur : %v", err)
	}

	fmt.Println("--------------------------------------------------")
	fmt.Println("✅ DATABASE READY (CARRIER-GRADE SEED)")
	fmt.Printf("🏢 Tenant ID : %s\n", testTenantID)
	fmt.Printf("👤 Username  : yvan_clean\n")
	fmt.Printf("🔑 Password  : %s\n", password)
	fmt.Printf("📊 Quota     : 10 GB\n")
	fmt.Println("--------------------------------------------------")
}
