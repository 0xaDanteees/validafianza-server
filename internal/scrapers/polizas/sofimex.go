package polizas

import (
	"regexp"

	"golang.org/x/net/html"
	"validafianza-server/internal/scrapers"
)

func NewSofimexScraper() *aspxPolizaScraper {
	pattern := regexp.MustCompile(`(?i)no se pudo validar`)
	return &aspxPolizaScraper{
		BaseScraper: scrapers.NewBaseScraper("SOFIMEX", "native"),
		baseURL:     "https://apps.sofimex.mx/valida_fianza_sofimex/Sitios/Default.aspx",
		buttonValue: "VALIDAR",
		isValid: func(doc *html.Node) bool {
			return !searchText(doc, pattern)
		},
	}
}
