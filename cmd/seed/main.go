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
	// 1. Charger le .env
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Erreur lors du chargement du fichier .env")
	}

	// 2. Connexion à Neon
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Impossible de se connecter à Neon: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	// IDs de test fixes pour éviter les erreurs de clés étrangères
	const (
		testTenantID = "550e8400-e29b-41d4-a716-446655440000"
		testUserID   = "660f8400-e29b-41d4-a716-446655441111"
	)

	fmt.Println("⏳ Initialisation des données de test...")

	// 3. Création du Tenant (Organisation)
	// On ajoute 'type' pour satisfaire la contrainte NOT NULL
	tenantQuery := `
		INSERT INTO tenants (id, name, type, created_at) 
		VALUES ($1, 'Nexora Test Corp', 'test', NOW())
		ON CONFLICT (id) DO NOTHING;
	`
	_, err = conn.Exec(context.Background(), tenantQuery, testTenantID)
	if err != nil {
		log.Fatalf("❌ Échec création tenant : %v", err)
	}
	fmt.Println("✅ Tenant prêt ou déjà existant.")

	// 4. Préparation du Hash propre
	password := "password123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	// 5. Création de l'utilisateur admin
	// On utilise ON CONFLICT pour pouvoir relancer le script sans erreur
	userQuery := `
		INSERT INTO users (id, username, password_hash, tenant_id, role, created_at) 
		VALUES ($1, $2, $3, $4, 'admin', NOW())
		ON CONFLICT (username) DO UPDATE SET password_hash = $3;
	`
	_, err = conn.Exec(context.Background(), userQuery, testUserID, "yvan_clean", string(hash), testTenantID)
	if err != nil {
		log.Fatalf("❌ Échec création utilisateur : %v", err)
	}

	fmt.Println("-----------------------------------------")
	fmt.Printf("🚀 VICTOIRE ! Données insérées avec succès.\n")
	fmt.Printf("👤 Utilisateur : yvan_clean\n")
	fmt.Printf("🔑 Password    : %s\n", password)
	fmt.Println("-----------------------------------------")
}
