package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"validafianza-server/internal/models"
	"validafianza-server/internal/repository"
	"validafianza-server/internal/scrapers/sanciones"
	"validafianza-server/internal/utils"
)

type SancionService struct {
	repo       *repository.SancionRepository
	scgScraper *sanciones.SCGScraper
	sfpScraper *sanciones.SFPScraper
	cacheTTL   int
}

func NewSancionService(repo *repository.SancionRepository, cacheTTL int) *SancionService {
	return &SancionService{
		repo:       repo,
		scgScraper: sanciones.NewSCGScraper(),
		sfpScraper: sanciones.NewSFPScraper(),
		cacheTTL:   cacheTTL,
	}
}

func (s *SancionService) ValidarRFCSCG(ctx context.Context, rfc string) (*models.SancionResultSCG, error) {
	if !utils.ValidarRFC(rfc) {
		return nil, fmt.Errorf("RFC no válido")
	}

	cacheKey := fmt.Sprintf("scg_%s", rfc)
	cached, err := s.repo.GetCached(ctx, cacheKey)
	if err == nil && cached != nil {
		var sanciones []models.SancionSCG
		if cached.Detalles != "" {
			json.Unmarshal([]byte(cached.Detalles), &sanciones)
		}

		return &models.SancionResultSCG{
			RFC:        cached.RFC,
			Sancionado: cached.Sancionado,
			Sanciones:  sanciones,
			Message:    getMessageSCG(cached.Sancionado),
			ValidadoEn: cached.CreatedAt,
		}, nil
	}

	sanciones, err := s.scgScraper.BuscarRFC(ctx, rfc)
	if err != nil {
		sanciones = []models.SancionSCG{}
	}

	sancionado := len(sanciones) > 0
	result := &models.SancionResultSCG{
		RFC:        rfc,
		Sancionado: sancionado,
		Sanciones:  sanciones,
		Message:    getMessageSCG(sancionado),
		ValidadoEn: time.Now(),
	}

	detallesJSON, _ := json.Marshal(sanciones)
	_ = s.repo.SaveCache(ctx, cacheKey, sancionado, string(detallesJSON), s.cacheTTL)

	return result, nil
}

func (s *SancionService) ValidarRFCSFP(ctx context.Context, rfc string) (*models.SancionResultSFP, error) {
	if !utils.ValidarRFC(rfc) {
		return nil, fmt.Errorf("RFC no válido")
	}

	cacheKey := fmt.Sprintf("sfp_%s", rfc)
	cached, err := s.repo.GetCached(ctx, cacheKey)
	if err == nil && cached != nil {
		var sanciones []models.SancionSFP
		if cached.Detalles != "" {
			json.Unmarshal([]byte(cached.Detalles), &sanciones)
		}

		return &models.SancionResultSFP{
			RFC:        cached.RFC,
			Sancionado: cached.Sancionado,
			Sanciones:  sanciones,
			Message:    getMessageSFP(cached.Sancionado),
			ValidadoEn: cached.CreatedAt,
		}, nil
	}

	sanciones, err := s.sfpScraper.BuscarRFC(ctx, rfc)
	if err != nil {
		sanciones = []models.SancionSFP{}
	}

	sancionado := len(sanciones) > 0
	result := &models.SancionResultSFP{
		RFC:        rfc,
		Sancionado: sancionado,
		Sanciones:  sanciones,
		Message:    getMessageSFP(sancionado),
		ValidadoEn: time.Now(),
	}

	detallesJSON, _ := json.Marshal(sanciones)
	_ = s.repo.SaveCache(ctx, cacheKey, sancionado, string(detallesJSON), s.cacheTTL)

	return result, nil
}

func getMessageSCG(sancionado bool) string {
	if sancionado {
		return "Proveedor SANCIONADO ante SCG (Contraloría CDMX)"
	}
	return "Proveedor NO sancionado ante SCG"
}

func getMessageSFP(sancionado bool) string {
	if sancionado {
		return "Proveedor SANCIONADO ante SFP (Función Pública)"
	}
	return "Proveedor NO sancionado ante SFP"
}
