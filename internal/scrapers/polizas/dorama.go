package polizas

import (
	"regexp"

	"golang.org/x/net/html"
	"validafianza-server/internal/scrapers"
)

func NewDoramaScraper() *aspxPolizaScraper {
	pattern := regexp.MustCompile(`(?i)Código de validación:`)
	return &aspxPolizaScraper{
		BaseScraper: scrapers.NewBaseScraper("DORAMA", "native"),
		baseURL:     "https://apps.dorama.mx/validafianza/sitios/Default.aspx",
		buttonValue: "Valida",
		isValid: func(doc *html.Node) bool {
			return searchText(doc, pattern)
		},
	}
}
