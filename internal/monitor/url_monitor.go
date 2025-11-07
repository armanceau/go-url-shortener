package monitor

import (
	"log"
	"context"
	"net/http"
	"sync" // Pour protéger l'accès concurrentiel à knownStates
	"time"

	_ "github.com/armanceau/go-url-shortener/internal/models"   // Importe les modèles de liens
	"github.com/armanceau/go-url-shortener/internal/repository" // Importe le repository de liens
)

// UrlMonitor gère la surveillance périodique des URLs longues.
type UrlMonitor struct {
	linkRepo    repository.LinkRepository // Pour récupérer les URLs à surveiller
	interval    time.Duration             // Intervalle entre chaque vérification (ex: 5 minutes)
	knownStates map[uint]bool             // État connu de chaque URL: map[LinkID]estAccessible (true/false)
	mu          sync.Mutex                // Mutex pour protéger l'accès concurrentiel à knownStates
}

//retourner instance UrlMonitor.
func NewUrlMonitor(linkRepo repository.LinkRepository, interval time.Duration) *UrlMonitor {
	return &UrlMonitor{
		linkRepo:    linkRepo,
		interval:    interval,
		knownStates: make(map[uint]bool),
	}
}

// Start lance la boucle de surveillance périodique des URLs.
func (m *UrlMonitor) Start() {
	log.Printf("[MONITOR] Démarrage du moniteur d'URLs avec un intervalle de %v...", m.interval)
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	m.checkUrls()

	for range ticker.C {
		m.checkUrls()
	}
}

// checkUrls effectue une vérification de l'état de toutes les URLs longues enregistrées.
func (m *UrlMonitor) checkUrls() {
	log.Println("[MONITOR] Lancement de la vérification de l'état des URLs...")

	//récupération de tout les liens méthode GetAllLinks
	links, err := m.linkRepo.GetAllLinks()
	if err != nil {
		log.Printf("[MONITOR] ERREUR lors de la récupération des liens pour la surveillance : %v", err)
		return
	}
	
	for _, link := range links {
		currentState :=  m.isUrlAccessible(link.LongURL)

		// Protéger l'accès à la map 'knownStates' car 'checkUrls' peut être exécuté concurremment
		m.mu.Lock()
		previousState, exists := m.knownStates[link.ID] // Récupère l'état précédent
		m.knownStates[link.ID] = currentState           // Met à jour l'état actuel
		m.mu.Unlock()

		// Si c'est la première vérification pour ce lien, on initialise l'état sans notifier.
		if !exists {
			log.Printf("[MONITOR] État initial pour le lien %s (%s) : %s",
				link.ShortCode, link.LongURL, formatState(currentState))
			continue
		}
		// Si l'état a changé, générer une fausse notification dans les logs.
		if previousState != currentState {
			log.Printf("[NOTIFICATION] Le lien %s (%s) est passé de %s à %s !",
				link.ShortCode, link.LongURL, formatState(previousState), formatState(currentState))
		}
	}
	log.Println("[MONITOR] Vérification de l'état des URLs terminée.")
}

// isUrlAccessible effectue une requête HTTP HEAD pour vérifier l'accessibilité d'une URL.
func (m *UrlMonitor) isUrlAccessible(url string) bool {
	//timeout 5sec
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	//tester l'accessibilité de la requete et gestion d'erreur par la méthode do.
	do := func(method string) (int, error) {
		req, err := http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			return 0, err
		}
		req.Header.Set("User-Agent", "go-url-shortener-monitor/1.0")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return 0, err
		}
		// ferme toujours le body
		if resp.Body != nil {
			resp.Body.Close()
		}
		return resp.StatusCode, nil
	}
	if code, err := do(http.MethodHead); err == nil {
		if code >= 200 && code < 400 {
			return true
		}
	} else {
		log.Printf("[MONITOR] Erreur HEAD '%s': %v (tentative GET)", url, err)
	}

	reqGet, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Printf("[MONITOR] Erreur création requête GET '%s': %v", url, err)
		return false
	}
	reqGet.Header.Set("User-Agent", "go-url-shortener-monitor/1.0")

	//appeler la méthode do pour GET
	if code, err := do(http.MethodGet); err == nil {
		return code >= 200 && code < 400
	} else {
		log.Printf("[MONITOR] Erreur GET '%s': %v", url, err)
		return false
	}
}

// formatState est une fonction utilitaire pour rendre l'état plus lisible dans les logs.
func formatState(accessible bool) string {
	if accessible {
		return "ACCESSIBLE"
	}
	return "INACCESSIBLE"
}
