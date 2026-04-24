package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"validafianza-server/internal/models"
)

type mockSancionService struct {
	validateSCGFunc func(ctx context.Context, rfc string) (*models.SancionResultSCG, error)
	validateSFPFunc func(ctx context.Context, rfc string) (*models.SancionResultSFP, error)
}

func (m *mockSancionService) ValidarRFCSCG(ctx context.Context, rfc string) (*models.SancionResultSCG, error) {
	if m.validateSCGFunc != nil {
		return m.validateSCGFunc(ctx, rfc)
	}
	return &models.SancionResultSCG{
		RFC:        rfc,
		Sancionado: false,
		Message:    "Proveedor NO sancionado ante SCG",
		ValidadoEn: time.Now(),
	}, nil
}

func (m *mockSancionService) ValidarRFCSFP(ctx context.Context, rfc string) (*models.SancionResultSFP, error) {
	if m.validateSFPFunc != nil {
		return m.validateSFPFunc(ctx, rfc)
	}
	return &models.SancionResultSFP{
		RFC:        rfc,
		Sancionado: false,
		Message:    "Proveedor NO sancionado ante SFP",
		ValidadoEn: time.Now(),
	}, nil
}

func TestValidarRFCSCG_Success_NoSancionado(t *testing.T) {
	mockService := &mockSancionService{
		validateSCGFunc: func(ctx context.Context, rfc string) (*models.SancionResultSCG, error) {
			return &models.SancionResultSCG{
				RFC:        rfc,
				Sancionado: false,
				Message:    "Proveedor NO sancionado ante SCG",
				ValidadoEn: time.Now(),
			}, nil
		},
	}

	handler := &SancionHandler{service: mockService}
	app := fiber.New()
	app.Post("/scg", handler.ValidarRFCSCG)

	reqBody := ValidarRFCRequest{RFC: "SIR130227V77"}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/scg", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var response SancionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success to be true")
	}
}

func TestValidarRFCSFP_Success_Sancionado(t *testing.T) {
	mockService := &mockSancionService{
		validateSFPFunc: func(ctx context.Context, rfc string) (*models.SancionResultSFP, error) {
			return &models.SancionResultSFP{
				RFC:        rfc,
				Sancionado: true,
				Sanciones: []models.SancionSFP{
					{
						RFC:               rfc,
						NumeroExpediente:  "EXP-001",
						NombreRazonSocial: "Test SA",
					},
				},
				Message:    "Proveedor SANCIONADO ante SFP",
				ValidadoEn: time.Now(),
			}, nil
		},
	}

	handler := &SancionHandler{service: mockService}
	app := fiber.New()
	app.Post("/sfp", handler.ValidarRFCSFP)

	reqBody := ValidarRFCRequest{RFC: "MASE810219AH5"}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/sfp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var response SancionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success to be true")
	}
}

func TestValidarRFCSCG_EmptyRFC(t *testing.T) {
	handler := &SancionHandler{service: &mockSancionService{}}
	app := fiber.New()
	app.Post("/scg", handler.ValidarRFCSCG)

	reqBody := ValidarRFCRequest{RFC: ""}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/scg", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	var response SancionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Error("Expected success to be false")
	}

	if response.Message != "RFC is required" {
		t.Errorf("Expected message 'RFC is required', got '%s'", response.Message)
	}
}
