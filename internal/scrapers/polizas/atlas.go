package polizas

import (
	"context"
	"fmt"
	"regexp"

	"validafianza-server/internal/models"
	"validafianza-server/internal/scrapers"
	"validafianza-server/internal/utils"
)

type AtlasScraper struct {
	*scrapers.BaseScraper
}

func NewAtlasScraper() *AtlasScraper {
	return &AtlasScraper{
		BaseScraper: scrapers.NewBaseScraper("FIANZAS_Y_CAUCIONES_ATLAS", "native"),
	}
}

func (s *AtlasScraper) Scrape(ctx context.Context, req *models.ScrapeRequest) (*models.ScrapeResult, error) {
	const baseURL = "https://www.fianzasatlas.net/efianza/Validador/ValidadorDocumentos.aspx"

	client, err := utils.NewASPXClient(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	formData, err := client.GetInitialState()
	if err != nil {
		return nil, fmt.Errorf("failed to get initial state: %w", err)
	}

	// Atlas uses different form fields than the standard ASPX pattern
	formData.Set("__EVENTTARGET", "")
	formData.Set("__EVENTARGUMENT", "")
	formData.Set("txtLineaValidacion", req.LookupKey)
	formData.Set("1", "RdTextoPDF")
	formData.Set("BtnValidarPDF", "Validar")

	doc, err := client.PostForm(formData)
	if err != nil {
		return nil, fmt.Errorf("failed to submit form: %w", err)
	}

	valid := searchText(doc, regexp.MustCompile(`(?i)La fianza fue validada exitosamente`))
	msg := "Póliza validada exitosamente en ATLAS"
	if !valid {
		msg = "No se pudo validar la póliza en ATLAS"
	}

	return &models.ScrapeResult{
		Valid:    valid,
		Invalid:  !valid,
		Message:  msg,
		Data:     map[string]interface{}{"provider": s.Name(), "lookup": req.LookupKey},
	}, nil
}
