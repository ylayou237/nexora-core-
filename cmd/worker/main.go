package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	// 1. Chargement de la config
	_ = godotenv.Load()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL manquante")
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		log.Fatalf("Impossible de se connecter à Neon: %v", err)
	}
	defer conn.Close(ctx)

	log.Println("🛠️ Worker de Maintenance des Partitions démarré...")

	// 2. Exécution de la maintenance
	maintainPartitions(ctx, conn)
}

func maintainPartitions(ctx context.Context, conn *pgx.Conn) {
	log.Println("⏳ Vérification des partitions d'audit...")

	// Appel de la fonction SQL que nous avons créée dans ta migration
	// Elle crée la partition du mois prochain si elle n'existe pas
	_, err := conn.Exec(ctx, "SELECT create_audit_partition_next_month();")
	if err != nil {
		log.Printf("❌ Erreur lors de la création de la partition : %v", err)
	} else {
		log.Println("✅ Partition du mois prochain vérifiée/créée.")
	}

	// Optionnel : Nettoyage des vieux logs (ex: plus de 7 ans comme défini dans ton SQL)
	_, err = conn.Exec(ctx, "SELECT archive_old_audit_logs();")
	if err != nil {
		log.Printf("❌ Erreur lors de l'archivage : %v", err)
	} else {
		log.Println("✅ Nettoyage des vieilles partitions terminé.")
	}

	log.Println("🚀 Maintenance terminée avec succès.")
}
