package cli

import (
	"fmt"
	"log"
	"net/url"
	"os"
	
	"github.com/spf13/cobra"
    "github.com/armanceau/go-url-shortener/cmd"
    "gorm.io/gorm"           
    "gorm.io/driver/sqlite"
	"github.com/armanceau/go-url-shortener/internal/repository"  
    "github.com/armanceau/go-url-shortener/internal/services"   
)

var longURLFlag string

// CreateCmd représente la commande 'create'
var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Crée une URL courte à partir d'une URL longue.",
	Long: `Cette commande raccourcit une URL longue fournie et affiche le code court généré.

Exemple:
  url-shortener create --url="https://www.google.com/search?q=go+lang"`,
	Run: func(cobraCmd *cobra.Command, args []string) {
		// verif flag existe

		if longURLFlag == "" {
			fmt.Println("Erreur: le flag est requis.")
			os.Exit(1)
		}
		//verifie si l'url est bien écrite
		_, err := url.ParseRequestURI(longURLFlag)
		if err != nil {
			fmt.Println("Erreur: l'URL n'est pas valide")
			os.Exit(1)
		}

		// Charger la configuration du fichier config.yaml
		cfg := cmd.Cfg
		if cfg == nil {
			fmt.Println("Erreur: impossible de charger la configuration")
			os.Exit(1)
		}

		// Init DB 
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
		
		// appel du  service pour créer le lien court
		link, err := linkService.CreateLink(longURLFlag)
		if err != nil {
			fmt.Printf("Erreur: impossible de créer le lien court: %v\n", err)
			os.Exit(1)
		}

		fullShortURL := fmt.Sprintf("%s/%s", cfg.Server.BaseURL, link.ShortCode)
		fmt.Printf("URL courte créée avec succès:\n")
		fmt.Printf("Code: %s\n", link.ShortCode)
		fmt.Printf("URL complète: %s\n", fullShortURL)
	},
}

// init() s'exécute automatiquement lors de l'importation du package.
// Il est utilisé pour définir les flags que cette commande accepte.

func init() {
	
 CreateCmd.Flags().StringVarP(&longURLFlag, "url", "u", "", "L'URL longue à raccourcir")
    
    // Marquer le flag comme requis
    CreateCmd.MarkFlagRequired("url")
    
    // Ajouter la commande à RootCmd
    cmd.RootCmd.AddCommand(CreateCmd)
}