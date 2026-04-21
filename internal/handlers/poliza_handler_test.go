package handlers_test

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"validafianza-server/internal/handlers"
	"validafianza-server/internal/models"
)

// ---------------------------------------------------------------------------
// Fake service
// ---------------------------------------------------------------------------

type fakeValidationService struct {
	result *models.ValidationResponse
	err    error
}

func (f *fakeValidationService) Validate(_ context.Context, _ *models.ValidationRequest) (*models.ValidationResponse, error) {
	return f.result, f.err
}

// ---------------------------------------------------------------------------
// Test app factory
// ---------------------------------------------------------------------------

func newTestApp(svc validationServicer) *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: fiber.DefaultErrorHandler})
	h := handlers.NewPolizaHandlerWithService(svc)
	app.Post("/api/v1/polizas/validate", h.ValidatePoliza)
	app.Get("/api/v1/polizas/providers", h.ListProviders)
	return app
}

// validationServicer matches the method set the handler needs.
type validationServicer interface {
	Validate(ctx context.Context, req *models.ValidationRequest) (*models.ValidationResponse, error)
}

func post(t *testing.T, app *fiber.App, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/v1/polizas/validate",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	_ = resp
	return nil // fiber Test returns *http.Response, not ResponseRecorder
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestValidatePoliza_InvalidJSON_Returns400(t *testing.T) {
	app := newTestApp(&fakeValidationService{})
	req := httptest.NewRequest("POST", "/api/v1/polizas/validate",
		strings.NewReader(`{not valid json`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "error") {
		t.Errorf("expected error field in body, got: %s", body)
	}
}

func TestValidatePoliza_ServiceError_Returns500(t *testing.T) {
	svc := &fakeValidationService{err: errors.New("db connection lost")}
	app := newTestApp(svc)

	req := httptest.NewRequest("POST", "/api/v1/polizas/validate",
		strings.NewReader(`{"provider_key":"MAPFRE","lookup_key":"POLIZA-001"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("status = %d, want 500", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "db connection lost") {
		t.Errorf("expected error message in body, got: %s", body)
	}
}

func TestValidatePoliza_ValidRequest_Returns200(t *testing.T) {
	svc := &fakeValidationService{
		result: &models.ValidationResponse{Valid: true, Message: "Póliza validada"},
	}
	app := newTestApp(svc)

	req := httptest.NewRequest("POST", "/api/v1/polizas/validate",
		strings.NewReader(`{"provider_key":"MAPFRE","lookup_key":"0032400003909000000491774"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "valido") {
		t.Errorf("expected 'valido' field in body, got: %s", body)
	}
}

func TestListProviders_Returns200WithAllProviders(t *testing.T) {
	app := newTestApp(&fakeValidationService{})

	req := httptest.NewRequest("GET", "/api/v1/polizas/providers", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	for _, provider := range []string{"MAPFRE", "BERKLEY_MEXICO_FIANZAS", "SOFIMEX", "TOKIO_MARINE"} {
		if !strings.Contains(string(body), provider) {
			t.Errorf("expected provider %q in response, got: %s", provider, body)
		}
	}
}
