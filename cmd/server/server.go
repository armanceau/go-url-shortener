package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	cmd2 "github.com/armanceau/go-url-shortener/cmd"
	"github.com/armanceau/go-url-shortener/internal/api"
	"github.com/armanceau/go-url-shortener/internal/models"
	"github.com/armanceau/go-url-shortener/internal/monitor"
	"github.com/armanceau/go-url-shortener/internal/repository"
	"github.com/armanceau/go-url-shortener/internal/services"
	"github.com/armanceau/go-url-shortener/internal/workers"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite" // Pure go SQLite driver, checkout https://github.com/glebarez/sqlite for details
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

// RunServerCmd représente la commande 'run-server' de Cobra.
// C'est le point d'entrée pour lancer le serveur de l'application.
var RunServerCmd = &cobra.Command{
	Use:   "server",
	Short: "Lance le serveur API de raccourcissement d'URLs et les processus de fond.",
	Long: `Cette commande initialise la base de données, configure les APIs,
démarre les workers asynchrones pour les clics et le moniteur d'URLs,
puis lance le serveur HTTP.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Créer une variable qui stock la configuration chargée globalement via cmd.cfg
		// Ne pas oublier la gestion d'erreur et faire un fatalF
		if cmd2.Cfg == nil {
			log.Fatalf("FATAL: Configuration not loaded")
		}
		cfg := cmd2.Cfg

		// Initialiser la connexion à la BDD
		db, err := gorm.Open(sqlite.Open(cfg.Database.Name), &gorm.Config{})
		if err != nil {
			log.Fatalf("FATAL: Échec de la connexion à la base de données: %v", err)
		}

		// Initialiser les repositories
		// Créez des instances de GormLinkRepository et GormClickRepository
		linkRepo := repository.NewLinkRepository(db)
		clickRepo := repository.NewClickRepository(db)

		// Laissez le log
		log.Println("Repositories initialisés.")

		// Initialiser les services métiers
		// Créez des instances de LinkService et ClickService, en leur passant les repositories nécessaires
		clickService := services.NewClickService(clickRepo)
		linkService := services.NewLinkService(linkRepo, clickService)

		// Laissez le log
		log.Println("Services métiers initialisés.")

		// Initialiser le channel ClickEventsChannel (api/handlers) des événements de clic et lancer les workers (StartClickWorkers)
		// Le channel est bufferisé avec la taille configurée
		// Passez le channel et le clickRepo aux workers
		cfg.ClickEventsChannel = make(chan models.ClickEvent, cfg.Analytics.BufferSize)
		workers.StartClickWorkers(cfg.Analytics.WorkerCount, cfg.ClickEventsChannel, clickRepo)

		// Remplacer les XXX par les bonnes variables
		log.Printf("Channel d'événements de clic initialisé avec un buffer de %d. %d worker(s) de clics démarré(s).",
			cfg.Analytics.BufferSize, cfg.Analytics.WorkerCount)

		// Initialiser et lancer le moniteur d'URLs
		// Utilisez l'intervalle configuré
		monitorInterval := time.Duration(cfg.Monitor.IntervalMinutes) * time.Minute
		urlMonitor := monitor.NewUrlMonitor(linkRepo, monitorInterval)

		// Lancez le moniteur dans sa propre goroutine
		go urlMonitor.Start()

		log.Printf("Moniteur d'URLs démarré avec un intervalle de %v.", monitorInterval)

		// Configurer le routeur Gin et les handlers API
		// Passez les services nécessaires aux fonctions de configuration des routes
		router := gin.Default()
		api.SetupRoutes(router, linkService, cfg)

		// Pas toucher au log
		log.Println("Routes API configurées.")

		// Créer le serveur HTTP Gin
		serverAddr := fmt.Sprintf(":%d", cfg.Server.Port)
		srv := &http.Server{
			Addr:    serverAddr,
			Handler: router,
		}

		// Démarrer le serveur Gin dans une goroutine anonyme pour ne pas bloquer
		go func() {
			log.Printf("Serveur HTTP démarré sur %s", serverAddr)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Erreur lors du démarrage du serveur: %v", err)
			}
		}()

		// Gérer l'arrêt propre du serveur (graceful shutdown)
		// Créez un channel pour les signaux OS (SIGINT, SIGTERM), bufferisé à 1
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM) // Attendre Ctrl+C ou signal d'arrêt

		// Bloquer jusqu'à ce qu'un signal d'arrêt soit reçu
		<-quit
		log.Println("Signal d'arrêt reçu. Arrêt du serveur...")

		// Arrêt propre du serveur HTTP avec un timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("Erreur lors de l'arrêt du serveur: %v", err)
		}

		// Donner un peu de temps aux workers pour finir et fermer les channels
		log.Println("Arrêt en cours... Donnez un peu de temps aux workers pour finir.")
		close(cfg.ClickEventsChannel)
		// Note: Le monitor n'a pas de méthode Stop, il s'arrêtera naturellement
		time.Sleep(2 * time.Second)

		log.Println("Serveur arrêté proprement.")
	},
}

func init() {
	// Ajouter la commande server au RootCmd
	cmd2.RootCmd.AddCommand(RunServerCmd)
}
