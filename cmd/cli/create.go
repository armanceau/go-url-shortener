package cli

import (
	"fmt"
	"log"
	"net/url"
	"os"

	cmd2 "github.com/armanceau/go-url-shortener/cmd"
	"github.com/armanceau/go-url-shortener/internal/repository"
	"github.com/armanceau/go-url-shortener/internal/services"
	"github.com/glebarez/sqlite" // Pure go SQLite driver
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

// longURLFlag stockera la valeur du flag --url
var longURLFlag string

// CreateCmd représente la commande 'create'
var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Crée une URL courte à partir d'une URL longue.",
	Long: `Cette commande raccourcit une URL longue fournie et affiche le code court généré.

Exemple:
  url-shortener create --url="https://www.google.com/search?q=go+lang"`,
	Run: func(cmd *cobra.Command, args []string) {
		// Valider que le flag --url a été fourni
		if longURLFlag == "" {
			log.Fatalf("FATAL: Le flag --url est requis")
		}

		// Validation basique du format de l'URL avec le package url et la fonction ParseRequestURI
		if _, err := url.ParseRequestURI(longURLFlag); err != nil {
			log.Printf("ERREUR: URL invalide: %v", err)
			os.Exit(1)
		}

		// Charger la configuration chargée globalement via cmd.cfg
		if cmd2.Cfg == nil {
			log.Fatalf("FATAL: Configuration not loaded")
		}

		// Initialiser la connexion à la base de données SQLite
		db, err := gorm.Open(sqlite.Open(cmd2.Cfg.Database.Name), &gorm.Config{})
		if err != nil {
			log.Fatalf("FATAL: Échec de la connexion à la base de données: %v", err)
		}

		sqlDB, err := db.DB()
		if err != nil {
			log.Fatalf("FATAL: Échec de l'obtention de la base de données SQL sous-jacente: %v", err)
		}

		// S'assurer que la connexion est fermée à la fin de l'exécution de la commande
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

		// Appeler le LinkService et la fonction CreateLink pour créer le lien court
		link, err := linkService.CreateLink(longURLFlag)
		if err != nil {
			log.Printf("ERREUR: Impossible de créer le lien court: %v", err)
			os.Exit(1)
		}

		fullShortURL := fmt.Sprintf("%s/%s", cmd2.Cfg.Server.BaseURL, link.ShortCode)
		fmt.Printf("URL courte créée avec succès:\n")
		fmt.Printf("Code: %s\n", link.ShortCode)
		fmt.Printf("URL complète: %s\n", fullShortURL)
	},
}

// init() s'exécute automatiquement lors de l'importation du package.
// Il est utilisé pour définir les flags que cette commande accepte.
func init() {
	// Définir le flag --url pour la commande create
	CreateCmd.Flags().StringVarP(&longURLFlag, "url", "u", "", "URL longue à raccourcir (requis)")

	// Marquer le flag comme requis
	if err := CreateCmd.MarkFlagRequired("url"); err != nil {
		log.Fatalf("FATAL: Impossible de marquer le flag url comme requis: %v", err)
	}

	// Ajouter la commande à RootCmd
	cmd2.RootCmd.AddCommand(CreateCmd)
}
