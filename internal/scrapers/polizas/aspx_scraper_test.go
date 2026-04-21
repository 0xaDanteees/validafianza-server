package polizas

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"validafianza-server/internal/models"
)

// aspxTestServer spins up a fake ASPX server.
// GET  → minimal page with hidden form inputs so GetInitialState() succeeds.
// POST → returns responseHTML so the scraper's isValid() can run.
func aspxTestServer(t *testing.T, responseHTML string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			fmt.Fprint(w, `<html><body>
				<input type="hidden" name="__VIEWSTATE" value="vs" />
				<input type="hidden" name="__VIEWSTATEGENERATOR" value="vsg" />
				<input type="hidden" name="__EVENTVALIDATION" value="ev" />
				<input type="submit" name="ctl00$maincontent$bValida" value="Validar" />
			</body></html>`)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, responseHTML)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func scrapeRequest(lookup string) *models.ScrapeRequest {
	return &models.ScrapeRequest{ProviderKey: "TEST", LookupKey: lookup}
}

func TestASPXScraperFlow(t *testing.T) {
	cases := []struct {
		name         string
		responseHTML string
		scraper      func(url string) *aspxPolizaScraper
		wantValid    bool
	}{
		{
			name:         "berkley/valid: VIGENTE present",
			responseHTML: `<html><body><span>Estado: VIGENTE</span></body></html>`,
			scraper: func(url string) *aspxPolizaScraper {
				s := NewBerkleyScraper(); s.baseURL = url; return s
			},
			wantValid: true,
		},
		{
			name:         "berkley/invalid: VIGENTE absent",
			responseHTML: `<html><body><p>Póliza no encontrada</p></body></html>`,
			scraper: func(url string) *aspxPolizaScraper {
				s := NewBerkleyScraper(); s.baseURL = url; return s
			},
			wantValid: false,
		},
		{
			name:         "sofimex/valid: error message absent",
			responseHTML: `<html><body><p>Póliza encontrada correctamente</p></body></html>`,
			scraper: func(url string) *aspxPolizaScraper {
				s := NewSofimexScraper(); s.baseURL = url; return s
			},
			wantValid: true,
		},
		{
			name:         "sofimex/invalid: no se pudo validar present",
			responseHTML: `<html><body><span>No se pudo validar la póliza solicitada</span></body></html>`,
			scraper: func(url string) *aspxPolizaScraper {
				s := NewSofimexScraper(); s.baseURL = url; return s
			},
			wantValid: false,
		},
		{
			name:         "dorama/valid: código de validación present",
			responseHTML: `<html><body><p>Código de validación: XYZ-001</p></body></html>`,
			scraper: func(url string) *aspxPolizaScraper {
				s := NewDoramaScraper(); s.baseURL = url; return s
			},
			wantValid: true,
		},
		{
			name:         "dorama/invalid: código absent",
			responseHTML: `<html><body><p>No se encontraron resultados</p></body></html>`,
			scraper: func(url string) *aspxPolizaScraper {
				s := NewDoramaScraper(); s.baseURL = url; return s
			},
			wantValid: false,
		},
		{
			name:         "tokio/valid: anote número present",
			responseHTML: `<html><body><strong>ANOTE ESTE NÚMERO DE VALIDACIÓN: 000999</strong></body></html>`,
			scraper: func(url string) *aspxPolizaScraper {
				s := NewTokioScraper(); s.baseURL = url; return s
			},
			wantValid: true,
		},
		{
			name:         "tokio/invalid: número absent",
			responseHTML: `<html><body><p>Póliza no vigente</p></body></html>`,
			scraper: func(url string) *aspxPolizaScraper {
				s := NewTokioScraper(); s.baseURL = url; return s
			},
			wantValid: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := aspxTestServer(t, tc.responseHTML)
			result, err := tc.scraper(srv.URL).Scrape(context.Background(), scrapeRequest("TEST-0001"))

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Valid != tc.wantValid {
				t.Errorf("Valid=%v wantValid=%v | msg=%q", result.Valid, tc.wantValid, result.Message)
			}
			if result.Valid == result.Invalid {
				t.Errorf("Valid and Invalid must be complementary, both are %v", result.Valid)
			}
			if result.Message == "" {
				t.Error("Message must never be empty")
			}
			if result.Data["provider"] == nil {
				t.Error("Data.provider must be set")
			}
		})
	}
}
