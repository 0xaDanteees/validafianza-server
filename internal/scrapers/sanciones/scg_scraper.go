package sanciones

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"validafianza-server/internal/models"
)

type SCGScraper struct {
	client *http.Client
	urls   []string
}

func NewSCGScraper() *SCGScraper {
	return &SCGScraper{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		urls: []string{
			"http://cgservicios.df.gob.mx/contraloria/consultaAdquisiciones.php",
			"http://cgservicios.df.gob.mx/contraloria/consultaObras.php",
			"http://cgservicios.df.gob.mx/contraloria/consultaPerfisicas.php",
		},
	}
}

func (s *SCGScraper) BuscarRFC(ctx context.Context, rfc string) ([]models.SancionSCG, error) {
	var allSanciones []models.SancionSCG

	for _, url := range s.urls {
		sanciones, err := s.extraerDatos(ctx, url)
		if err != nil {
			continue
		}
		allSanciones = append(allSanciones, sanciones...)
	}

	var matching []models.SancionSCG
	for _, sancion := range allSanciones {
		if strings.TrimSpace(sancion.RFC) == rfc {
			matching = append(matching, sancion)
		}
	}

	return matching, nil
}

func (s *SCGScraper) extraerDatos(ctx context.Context, url string) ([]models.SancionSCG, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("error parsing HTML: %w", err)
	}

	var sanciones []models.SancionSCG

	doc.Find("table[width='98%']").Each(func(i int, table *goquery.Selection) {
		sancion := models.SancionSCG{}
		
		table.Find("tr").Each(func(j int, row *goquery.Selection) {
			cells := row.Find("td")
			if cells.Length() > 1 {
				key := strings.TrimSpace(strings.Replace(cells.Eq(0).Text(), ":", "", -1))
				value := strings.TrimSpace(cells.Eq(1).Text())

				switch key {
				case "No.":
					sancion.No = value
				case "Expediente":
					sancion.Expediente = value
				case "Persona":
					sancion.Persona = value
				case "R.F.C.":
					sancion.RFC = value
				case "Fecha de Resolución":
					sancion.FechaResolucion = value
				case "Infracción":
					sancion.Infraccion = value
				case "Sanción":
					sancion.Sancion = value
				case "Circular":
					sancion.Circular = value
				case "Publicación en la Gaceta Oficial":
					sancion.Publicacion = value
				case "Vigencia":
					sancion.Vigencia = value
				}
			}
		})

		if sancion.RFC != "" {
			sanciones = append(sanciones, sancion)
		}
	})

	return sanciones, nil
}
