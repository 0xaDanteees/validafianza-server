package polizas

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"validafianza-server/internal/models"
	"validafianza-server/internal/scrapers"
	"validafianza-server/internal/utils"
)

type MapfreScraper struct {
	*scrapers.BaseScraper
}

func NewMapfreScraper() *MapfreScraper {
	return &MapfreScraper{
		BaseScraper: scrapers.NewBaseScraper("MAPFRE", "native"),
	}
}

func (s *MapfreScraper) Scrape(ctx context.Context, req *models.ScrapeRequest) (*models.ScrapeResult, error) {
	const baseURL = "https://validacionfianzas.mapfre.com.mx/ValidaMapfre/Sitios/Default.aspx"
	client, err := utils.NewASPXClient(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	formData, err := client.GetInitialState()
	if err != nil {
		return nil, fmt.Errorf("Failed to get initial state: %w", err)
	}

	formData.Set(
		"ctl00$maincontent$ap1_content$lvCompleto",
		req.LookupKey,
	)
	formData.Set(
		"ctl00$maincontent$bValida",
		"Validar",
	)

	doc, err := client.PostForm(formData)
	if err != nil {
		return nil, fmt.Errorf("failed to submit form: %w", err)
	}

	hasError := s.hasErrorPattern(doc)
	htmlExtract := s.extractMainTable(doc)

	return &models.ScrapeResult{
		Valid:       !hasError,
		Invalid:     hasError,
		Message:     s.getMessage(hasError),
		HTMLExtract: htmlExtract,
        Data: map[string]interface{}{
            "provider": "MAPFRE",
            "lookup":   req.LookupKey,
        },
    }, nil
}

func (s *MapfreScraper) hasErrorPattern(doc *html.Node) bool {
	pattern := regexp.MustCompile(`(?i)no se pudo validar`)
	return searchText(doc, pattern)
}

func (s *MapfreScraper) getMessage(hasError bool) string {
	if hasError {
		return "No se pudo validar la póliza en MAPFRE"
	}
	return "Póliza validada exitosamente"
}

func (s *MapfreScraper) extractMainTable(doc *html.Node) []string {
	var result []string
	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "table" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "mainTable") {
					result = append(result, renderNode(n))
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	extract(doc)
	return result
}



