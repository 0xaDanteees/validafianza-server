package polizas

import (
	"regexp"

	"golang.org/x/net/html"
	"validafianza-server/internal/scrapers"
)

func NewBerkleyScraper() *aspxPolizaScraper {
	pattern := regexp.MustCompile(`(?i)VIGENTE`)
	return &aspxPolizaScraper{
		BaseScraper: scrapers.NewBaseScraper("BERKLEY_MEXICO_FIANZAS", "native"),
		baseURL:     "https://validafianza.berkleymex.com/ValidaFianzaBerkley/sitios/Default.aspx",
		buttonValue: "Validar",
		isValid: func(doc *html.Node) bool {
			return searchText(doc, pattern)
		},
	}
}
