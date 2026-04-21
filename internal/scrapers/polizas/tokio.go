package polizas

import (
	"regexp"

	"golang.org/x/net/html"
	"validafianza-server/internal/scrapers"
)

func NewTokioScraper() *aspxPolizaScraper {
	pattern := regexp.MustCompile(`(?i)ANOTE ESTE NÚMERO DE VALIDACIÓN`)
	return &aspxPolizaScraper{
		BaseScraper: scrapers.NewBaseScraper("TOKIO_MARINE", "native"),
		baseURL:     "https://dynamic.tmhcc.com/validation/Sitios/Default.aspx",
		buttonValue: "VALIDAR",
		isValid: func(doc *html.Node) bool {
			return searchText(doc, pattern)
		},
	}
}
