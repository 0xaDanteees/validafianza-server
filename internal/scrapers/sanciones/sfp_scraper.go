package sanciones

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"validafianza-server/internal/models"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

type SFPScraper struct {
	dataCache     []SFPEntidad
	lastScrapedAt time.Time
	cacheValidFor time.Duration
}

type SFPEntidad struct {
	NumeroExpediente  string   `json:"numero_expediente"`
	NombreRazonSocial string   `json:"nombre_razon_social"`
	RFC               string   `json:"rfc"`
	LeyesInfringidas  string   `json:"leyes_infringidas"`
	Multa             SFPMulta `json:"multa"`
	Plazo             SFPPlazo `json:"plazo"`
}

type SFPMulta struct {
	MontoCF string `json:"monto_cf"`
}

type SFPPlazo struct {
	PlazoInha interface{} `json:"plazo_inha"`
}

func NewSFPScraper() *SFPScraper {
	return &SFPScraper{
		cacheValidFor: 24 * time.Hour,
	}
}

func (s *SFPScraper) BuscarRFC(ctx context.Context, rfc string) ([]models.SancionSFP, error) {
	if time.Since(s.lastScrapedAt) > s.cacheValidFor || len(s.dataCache) == 0 {
		if err := s.scrapeData(ctx); err != nil {
			return nil, fmt.Errorf("error scraping SFP data: %w", err)
		}
	}

	var matching []models.SancionSFP
	for _, entidad := range s.dataCache {
		if strings.Contains(strings.ToUpper(entidad.RFC), strings.ToUpper(rfc)) {
			plazoStr := "No especificado"
			if entidad.Plazo.PlazoInha != nil {
				if str, ok := entidad.Plazo.PlazoInha.(string); ok {
					plazoStr = str
				}
			}

			matching = append(matching, models.SancionSFP{
				NumeroExpediente:  entidad.NumeroExpediente,
				NombreRazonSocial: entidad.NombreRazonSocial,
				RFC:               entidad.RFC,
				LeyesInfringidas:  entidad.LeyesInfringidas,
				MontoMulta:        entidad.Multa.MontoCF,
				PlazoInha:         plazoStr,
			})
		}
	}

	return matching, nil
}

func (s *SFPScraper) scrapeData(ctx context.Context) error {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	taskCtx, cancel = context.WithTimeout(taskCtx, 120*time.Second)
	defer cancel()

	var htmlContent string

	err := chromedp.Run(taskCtx,
		network.Enable(),
		chromedp.Navigate("https://directoriosancionados.apps.funcionpublica.gob.mx/"),
		chromedp.WaitVisible(`select[name="cfiltro"]`, chromedp.ByQuery),
		chromedp.SetValue(`select[name="cfiltro"]`, "Todos", chromedp.ByQuery),
		chromedp.Sleep(5*time.Second),
		chromedp.WaitVisible(`table.table`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		chromedp.OuterHTML(`html`, &htmlContent, chromedp.ByQuery),
	)

	if err != nil {
		return fmt.Errorf("chromedp error: %w", err)
	}

	entidades, err := s.parseHTMLTable(htmlContent)
	if err != nil {
		return fmt.Errorf("error parsing HTML: %w", err)
	}

	s.dataCache = entidades
	s.lastScrapedAt = time.Now()

	slog.Info("SFP data scraped successfully", "count", len(entidades))
	return nil
}

func (s *SFPScraper) parseHTMLTable(htmlContent string) ([]SFPEntidad, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	var entidades []SFPEntidad

	doc.Find("table.table tbody tr").Each(func(i int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() >= 6 {
			entidad := SFPEntidad{
				NumeroExpediente:  strings.TrimSpace(cells.Eq(0).Text()),
				NombreRazonSocial: strings.TrimSpace(cells.Eq(1).Text()),
				RFC:               strings.TrimSpace(cells.Eq(2).Text()),
				LeyesInfringidas:  strings.TrimSpace(cells.Eq(3).Text()),
				Multa: SFPMulta{
					MontoCF: strings.TrimSpace(cells.Eq(4).Text()),
				},
				Plazo: SFPPlazo{
					PlazoInha: strings.TrimSpace(cells.Eq(5).Text()),
				},
			}
			if entidad.RFC != "" {
				entidades = append(entidades, entidad)
			}
		}
	})

	return entidades, nil
}
