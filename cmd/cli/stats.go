package cli

import (
	"fmt"
	"log"
	"os"

	cmd2 "github.com/armanceau/go-url-shortener/cmd"
	"github.com/armanceau/go-url-shortener/internal/repository"
	"github.com/armanceau/go-url-shortener/internal/services"
	"github.com/spf13/cobra"

	"github.com/glebarez/sqlite" // Pure go SQLite driver
	"gorm.io/gorm"
)

// shortCodeFlag stockera la valeur du flag --code
var shortCodeFlag string

// StatsCmd représente la commande 'stats'
var StatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Affiche les statistiques (nombre de clics) pour un lien court.",
	Long: `Cette commande permet de récupérer et d'afficher le nombre total de clics
pour une URL courte spécifique en utilisant son code.

Exemple:
  url-shortener stats --code="xyz123"`,
	Run: func(cmd *cobra.Command, args []string) {
		// Valider que le flag --code a été fourni
		if shortCodeFlag == "" {
			log.Printf("ERREUR: Le flag --code est requis")
			os.Exit(1)
		}

		// Charger la configuration chargée globalement via cmd.cfg
		if cmd2.Cfg == nil {
			log.Fatalf("FATAL: Configuration not loaded")
		}

		// Initialiser la connexion à la BDD
		db, err := gorm.Open(sqlite.Open(cmd2.Cfg.Database.Name), &gorm.Config{})
		if err != nil {
			log.Fatalf("FATAL: Échec de la connexion à la base de données: %v", err)
		}

		sqlDB, err := db.DB()
		if err != nil {
			log.Fatalf("FATAL: Échec de l'obtention de la base de données SQL sous-jacente: %v", err)
		}

		// S'assurer que la connexion est fermée à la fin de l'exécution de la commande grâce à defer
		defer func() {
			if err := sqlDB.Close(); err != nil {
				log.Printf("Erreur lors de la fermeture de la base de données: %v", err)
			}
		}()

		// Initialiser les repositories et services nécessaires NewLinkRepository & NewLinkService
		linkRepo := repository.NewLinkRepository(db)
		clickRepo := repository.NewClickRepository(db)
		clickService := services.NewClickService(clickRepo)
		linkService := services.NewLinkService(linkRepo, clickService)

		// Appeler GetLinkStats pour récupérer le lien et ses statistiques
		// Attention, la fonction retourne 3 valeurs
		link, totalClicks, err := linkService.GetLinkStats(shortCodeFlag)
		if err != nil {
			log.Printf("ERREUR: Impossible de récupérer les statistiques pour le code '%s': %v", shortCodeFlag, err)
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
	StatsCmd.Flags().StringVarP(&shortCodeFlag, "code", "c", "", "Code court pour lequel récupérer les statistiques (requis)")

	// Marquer le flag comme requis
	if err := StatsCmd.MarkFlagRequired("code"); err != nil {
		log.Fatalf("FATAL: Impossible de marquer le flag code comme requis: %v", err)
	}

	// Ajouter la commande à RootCmd
	cmd2.RootCmd.AddCommand(StatsCmd)
}
