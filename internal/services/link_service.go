package services

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"

	"gorm.io/gorm"

	"github.com/armanceau/go-url-shortener/internal/models"
	"github.com/armanceau/go-url-shortener/internal/repository"
)

// Définition du jeu de caractères pour la génération des codes courts.
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// LinkService est une structure qui fournit des méthodes pour la logique métier des liens.
type LinkService struct {
	linkRepo     repository.LinkRepository
	clickService *ClickService
}

// NewLinkService crée et retourne une nouvelle instance de LinkService.
func NewLinkService(linkRepo repository.LinkRepository, clickService *ClickService) *LinkService {
	return &LinkService{
		linkRepo:     linkRepo,
		clickService: clickService,
	}
}

// GenerateShortCode génère un code court aléatoire d'une longueur spécifiée.
func (s *LinkService) GenerateShortCode(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("length must be positive")
	}

	result := make([]byte, length)
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %w", err)
		}
		result[i] = charset[num.Int64()]
	}

	return string(result), nil
}

// CreateLink crée un nouveau lien raccourci.
// Il génère un code court unique, puis persiste le lien dans la base de données.
func (s *LinkService) CreateLink(longURL string) (*models.Link, error) {
	var shortCode string
	const maxRetries = 5

	for i := 0; i < maxRetries; i++ {
		// Génère un code de 6 caractères
		code, err := s.GenerateShortCode(6)
		if err != nil {
			return nil, fmt.Errorf("failed to generate short code: %w", err)
		}

		// Vérifie si le code existe déjà en base de données
		_, err = s.GetLinkByShortCode(code)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				shortCode = code
				break
			}
			return nil, fmt.Errorf("database error checking short code uniqueness: %w", err)
		}

		log.Printf("Short code '%s' already exists, retrying generation (%d/%d)...", code, i+1, maxRetries)
	}

	// Vérifie si un code unique a été trouvé
	if shortCode == "" {
		return nil, errors.New("failed to generate unique short code after maximum retries")
	}

	// Création du nouveau lien
	link := &models.Link{
		ShortCode: shortCode,
		LongURL:   longURL,
	}

	// Persiste le nouveau lien dans la base de données via le repository
	err := s.linkRepo.CreateLink(link)
	if err != nil {
		return nil, fmt.Errorf("failed to create link in database: %w", err)
	}
	return link, nil
}

// GetLinkByShortCode récupère un lien via son code court.
func (s *LinkService) GetLinkByShortCode(shortCode string) (*models.Link, error) {
	link, err := s.linkRepo.GetLinkByShortCode(shortCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("link with short code '%s' not found", shortCode)
		}
		return nil, err
	}
	return link, nil
}

// GetLinkStats récupère les statistiques pour un lien donné (nombre total de clics).
// Il interagit avec le LinkRepository pour obtenir le lien, puis avec le ClickRepository
func (s *LinkService) GetLinkStats(shortCode string) (*models.Link, int, error) {
	// Récupérer le lien par son shortCode
	link, err := s.GetLinkByShortCode(shortCode)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get link: %w", err)
	}

	// Compter le nombre de clics pour ce LinkID
	clickCount, err := s.clickService.GetClicksCountByLinkID(link.ID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count clicks: %w", err)
	}

	return link, clickCount, nil
}
