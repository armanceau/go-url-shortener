package cli

import (
	"fmt"
	"log"
	"os"

	cmd2 "github.com/armanceau/go-url-shortener/cmd"
	"github.com/armanceau/go-url-shortener/internal/repository"
	"github.com/armanceau/go-url-shortener/internal/services"
	"github.com/spf13/cobra"

	"gorm.io/driver/sqlite" // Driver SQLite pour GORM
	"gorm.io/gorm"
)

var shortCodeFlag string

// StatsCmd représente la commande 'stats'
var StatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Affiche les statistiques (nombre de clics) pour un lien court.",
	Long: `Cette commande permet de récupérer et d'afficher le nombre total de clics
pour une URL courte spécifique en utilisant son code.

Exemple:
  url-shortener stats --code="xyz123"`,
	Run: func(cobraCmd *cobra.Command, args []string) {

		if shortCodeFlag == "" {
			fmt.Println("Erreur: Le flag --code est requis.")
			os.Exit(1)
		}

		cfg := cmd2.Cfg
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

		linkRepo := repository.NewLinkRepository(db)
		clickRepo := repository.NewClickRepository(db)
		clickService := services.NewClickService(clickRepo)
		linkService := services.NewLinkService(linkRepo, clickService)

		// Appeler GetLinkStats pour récupérer le lien et ses statistiques
		link, totalClicks, err := linkService.GetLinkStats(shortCodeFlag)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				fmt.Printf("Erreur: aucun lien trouvé avec le code '%s'\n", shortCodeFlag)
			} else {
				fmt.Printf("Erreur: impossible de récupérer les statistiques: %v\n", err)
			}
			os.Exit(1)
		}

		fmt.Printf("Statistiques pour le code court: %s\n", link.ShortCode)
		fmt.Printf("URL longue: %s\n", link.LongURL)
		fmt.Printf("Total de clics: %d\n", totalClicks)
	},
}

// init() s'exécute automatiquement lors de l'importation du package.
// Il est utilisé pour définir les flags que cette commande accepte.

func init() {
    // Définir le flag --code pour la commande stats
    StatsCmd.Flags().StringVarP(&shortCodeFlag, "code", "c", "", "Le code court du lien")
    
    // Marquer le flag comme requis
    StatsCmd.MarkFlagRequired("code")
    
    // Ajouter la commande à RootCmd
    cmd2.RootCmd.AddCommand(StatsCmd)
}
