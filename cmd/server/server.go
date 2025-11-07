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

	"github.com/armanceau/go-url-shortener/cmd"
	"github.com/armanceau/go-url-shortener/internal/api"
	"github.com/armanceau/go-url-shortener/internal/monitor"
	"github.com/armanceau/go-url-shortener/internal/repository"
	"github.com/armanceau/go-url-shortener/internal/services"
	"github.com/armanceau/go-url-shortener/internal/workers"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// RunServerCmd représente la commande 'run-server' de Cobra.
// C'est le point d'entrée pour lancer le serveur de l'application.
var RunServerCmd = &cobra.Command{
	Use:   "run-server",
	Short: "Lance le serveur API de raccourcissement d'URLs et les processus de fond.",
	Long: `Cette commande initialise la base de données, configure les APIs,
démarre les workers asynchrones pour les clics et le moniteur d'URLs,
puis lance le serveur HTTP.`,
	Run: func(cobraCmd *cobra.Command, args []string) {
		
		// Ne pas oublier la gestion d'erreur et faire un fatalF
		cfg := cmd.Cfg
		if cfg == nil {
			log.Fatalf("FATAL: impossible de charger la configuration")
		}

		db, err := gorm.Open(sqlite.Open(cfg.Database.Name), &gorm.Config{})
		if err != nil {
			log.Fatalf("FATAL: Échec de la connexion à la base de données: %v", err)
		}

		// Créez des instances de GormLinkRepository et GormClickRepository.
		linkRepo := repository.NewLinkRepository(db)
		clickRepo := repository.NewClickRepository(db)

		// Laissez le log
		log.Println("Repositories initialisés.")

		// Créez des instances de LinkService et ClickService, en leur passant les repositories nécessaires.
		clickService := services.NewClickService(clickRepo)
		linkService := services.NewLinkService(linkRepo, clickService)

		// Laissez le log
		log.Println("Services métiers initialisés.")

		
		// Le channel est bufferisé avec la taille configurée.
		// Passez le channel et le clickRepo aux workers.
		api.ClickEventsChannel = make(chan api.ClickEvent, cfg.Analytics.BufferSize)
		workers.StartClickWorkers(api.ClickEventsChannel, clickRepo, cfg.Analytics.WorkerCount)

		log.Printf("Channel d'événements de clic initialisé avec un buffer de %d. %d worker(s) de clics démarré(s).",
			cfg.Analytics.BufferSize, cfg.Analytics.WorkerCount)

		// Utilisez l'intervalle configuré
		monitorInterval := time.Duration(cfg.Monitor.IntervalMinutes) * time.Minute
		urlMonitor := monitor.NewUrlMonitor(linkRepo, monitorInterval) // Le moniteur a besoin du linkRepo et de l'interval

		go urlMonitor.Start()

		log.Printf("Moniteur d'URLs démarré avec un intervalle de %v.", monitorInterval)

		// Passez les services nécessaires aux fonctions de configuration des routes.
		router := gin.Default()
		api.SetupRoutes(router, linkService, clickService)

		// Pas toucher au log
		log.Println("Routes API configurées.")

		// Créer le serveur HTTP Gin
		serverAddr := fmt.Sprintf(":%d", cfg.Server.Port)
		srv := &http.Server{
			Addr:    serverAddr,
			Handler: router,
		}

		
		// Pensez à logger des ptites informations...
		go func() {
			log.Printf("Démarrage du serveur HTTP sur %s", serverAddr)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("FATAL: Erreur lors du démarrage du serveur: %v", err)
			}
		}()

		// Gére l'arrêt propre du serveur (graceful shutdown).
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM) // Attendre Ctrl+C ou signal d'arrêt

		// Bloquer jusqu'à ce qu'un signal d'arrêt soit reçu.
		<-quit
		log.Println("Signal d'arrêt reçu. Arrêt du serveur...")

		// Arrêt propre du serveur HTTP avec un timeout.
		log.Println("Arrêt en cours... Donnez un peu de temps aux workers pour finir.")
		time.Sleep(5 * time.Second)

		log.Println("Serveur arrêté proprement.")
	},
}

func init() {

	cmd.RootCmd.AddCommand(RunServerCmd)
}
