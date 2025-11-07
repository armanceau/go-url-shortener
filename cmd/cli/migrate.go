package cli

import (
	"fmt"
	"log"
	"os"

	"github.com/armanceau/go-url-shortener/cmd"
	"github.com/armanceau/go-url-shortener/internal/models" // ← Ajoute cette ligne
	"github.com/spf13/cobra"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	// Driver SQLite pour GORM
)

// MigrateCmd représente la commande 'migrate'
var MigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Exécute les migrations de la base de données pour créer ou mettre à jour les tables.",
	Long: `Cette commande se connecte à la base de données configurée (SQLite)
et exécute les migrations automatiques de GORM pour créer les tables 'links' et 'clicks'
basées sur les modèles Go.`,
	Run: func(cobraCmd *cobra.Command, args []string) {

		// meme princie que create.go
		cfg := cmd.Cfg
		if cfg == nil {
			fmt.Println("Erreur: impossible de charger la configuration")
			os.Exit(1)
		}

		db, err := gorm.Open(sqlite.Open(cfg.Database.Name), &gorm.Config{})
		if err != nil {
			log.Fatalf("FATAL: Échec de la connexion à la base de données: %v", err)
		}

		sqlDB, err := db.DB()
		if err != nil {
			log.Fatalf("FATAL: Échec de l'obtention de la base de données SQL sous-jacente: %v", err)
		}
		defer sqlDB.Close()

		err = db.AutoMigrate(&models.Link{}, &models.Click{})
		if err != nil {
			log.Fatalf("FATAL: Échec des migrations: %v", err)
		}

		fmt.Println("Migrations de la base de données exécutées avec succès.")
	},
}

func init() {
	// Ajoute la commande à RootCmd
	cmd.RootCmd.AddCommand(MigrateCmd)
}
