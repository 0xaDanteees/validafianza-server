package polizas

import (
	"context"
	"fmt"

	"golang.org/x/net/html"
	"validafianza-server/internal/models"
	"validafianza-server/internal/scrapers"
	"validafianza-server/internal/utils"
)

type aspxPolizaScraper struct {
	*scrapers.BaseScraper
	baseURL     string
	buttonValue string
	isValid     func(*html.Node) bool
}

func (s *aspxPolizaScraper) Scrape(ctx context.Context, req *models.ScrapeRequest) (*models.ScrapeResult, error) {
	client, err := utils.NewASPXClient(s.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	formData, err := client.GetInitialState()
	if err != nil {
		return nil, fmt.Errorf("failed to get initial state: %w", err)
	}

	formData.Set("ctl00$maincontent$ap1_content$lvCompleto", req.LookupKey)
	formData.Set("ctl00$maincontent$bValida", s.buttonValue)

	doc, err := client.PostForm(formData)
	if err != nil {
		return nil, fmt.Errorf("failed to submit form: %w", err)
	}

	valid := s.isValid(doc)
	msg := "Póliza validada exitosamente en " + s.Name()
	if !valid {
		msg = "No se pudo validar la póliza en " + s.Name()
	}

	return &models.ScrapeResult{
		Valid:    valid,
		Invalid:  !valid,
		Message:  msg,
		Data:     map[string]interface{}{"provider": s.Name(), "lookup": req.LookupKey},
	}, nil
}
