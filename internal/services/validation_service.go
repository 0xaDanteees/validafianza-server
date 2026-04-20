package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"validafianza-server/internal/models"
	"validafianza-server/internal/repository"
	"validafianza-server/internal/scrappers"
)


type ValidationService struct {
	repo		*repository.PostgresRepo
	cacheRepo	*repository.CacheRepo
	CacheTTl	*time.Duration
}

func NewValidationService(
	repo			*repository.PostgresRepo,
	cache			*repository.CacheRepo,
	cacheTTLHours 	int
) *ValidationService {
	return &ValidationService{
		repo:	repo,
		cacheRepo: cacheRepo,
		cacheTTL: time.Duration(cacheTTLHours) * time.Hour,
	}
}


func (s *ValidationService) Validate(ctx context.Context, req *models.ValidationRequest)
(*models.ValidationResponse, error) {
	provider, err := s.repo.GetProviderByKey(ctx, req.ProviderKey)
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

	if !provider.IsActive {
		return &models.ValidationResponse{
			Valid: false,
			InValid: true,
			Message: "Provicer temporarily unavailable",
		}, nil
	}

	lookupHash := s.generateLookupHash(provider.ID, req.LookupKey)

	if !req.ForceRefresh {
		cached, err := s.cacheRepo.Get(ctx, lookupHash)
		if err == nil && cached != nil {
			return cached, nil
		}
	}

	scraper, err := scrapers.Get(provider.ProviderKey)

	if err != nil {
		return nil, fmt.Errorf("scraper not found: %w", err)
	}

	scrapeReq := &models.ScrapeRequest{
		ProviderKey: provider.ProviderKey,
		LookupKey: req.LookupKey,
		Config: provider.Config,
	}

	result, err := scraper.Scrape(ctx, scrapeReq)

	if err != nil {
		return nil, fmt.Errorf("scraping failed: %w", err)
	}

	now := time.Now()
	response := &models.ValidationResponse{
        Valid:       result.Valid,
        Invalid:     result.Invalid,
        Message:     result.Message,
        Data:        result.Data,
        HTMLExtract: result.HTMLExtract,
        ScrapedAt:   &now,
    }

	if err := s.cacherepo.Set(ctx, lookupHash, provider.ID, req.LookupKey, response, s.cacheTTL);
	err != nil {
		fmt.Printf("cache write failed: %v\n", err)
	}

	return response, nil
}

func (s *ValidationService) generateLookupHash(providerID int32, lookupKey string) string {
	data := fmt.Sprintf("%d%s", providerID, lookupKey)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}