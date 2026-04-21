package polizas

import (
	"context"
	"fmt"
	"regexp"

	"validafianza-server/internal/models"
	"validafianza-server/internal/scrapers"
	"validafianza-server/internal/utils"
)

type AsertaScraper struct {
	*scrapers.BaseScraper
}

func NewAsertaScraper() *AsertaScraper {
	return &AsertaScraper{
		BaseScraper: scrapers.NewBaseScraper("ASERTA_FIANZAS_SEGUROS", "native"),
	}
}

func (s *AsertaScraper) Scrape(ctx context.Context, req *models.ScrapeRequest) (*models.ScrapeResult, error) {
	const baseURL = "https://fianzas.aserta.com.mx/misFianzasAsertaIP17/resumenFianzaLinea.jsp"

	doc, err := utils.FetchJSPPage(ctx, baseURL, req.LookupKey)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}

	valid := !searchText(doc, regexp.MustCompile(`(?i)no existe`))
	msg := "Póliza validada exitosamente en ASERTA"
	if !valid {
		msg = "No se pudo validar la póliza en ASERTA"
	}

	return &models.ScrapeResult{
		Valid:    valid,
		Invalid:  !valid,
		Message:  msg,
		Data:     map[string]interface{}{"provider": s.Name(), "lookup": req.LookupKey},
	}, nil
}
