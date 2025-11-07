package services

import (
	"fmt"

	"github.com/armanceau/go-url-shortener/internal/models"
	"github.com/armanceau/go-url-shortener/internal/repository" // Importe le package repository
)

// ClickService est une structure qui fournit des méthodes pour la logique métier des clics.
type ClickService struct {
	clickRepo repository.ClickRepository
}

// NewClickService crée et retourne une nouvelle instance de ClickService.
func NewClickService(clickRepo repository.ClickRepository) *ClickService {
	return &ClickService{
		clickRepo: clickRepo,
	}
}

// RecordClick enregistre un nouvel événement de clic dans la base de données.
func (s *ClickService) RecordClick(click *models.Click) error {
	err := s.clickRepo.CreateClick(click)
	if err != nil {
		return fmt.Errorf("failed to create click in database: %w", err)
	}
	return nil
}

// GetClicksCountByLinkID récupère le nombre total de clics pour un LinkID donné.
// Cette méthode pourrait être utilisée par le LinkService pour les statistiques, ou directement par l'API stats.
func (s *ClickService) GetClicksCountByLinkID(linkID uint) (int, error) {
	count, err := s.clickRepo.CountClicksByLinkID(linkID)
	if err != nil {
		return 0, fmt.Errorf("failed to count clicks for linkID %d: %w", linkID, err)
	}
	return count, nil
}
