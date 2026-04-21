package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"validafianza-server/internal/models"
	"validafianza-server/internal/services"
)

type validationServicer interface {
	Validate(ctx context.Context, req *models.ValidationRequest) (*models.ValidationResponse, error)
}

type PolizaHandler struct {
	validationService validationServicer
}

func NewPolizaHandler(vs *services.ValidationService) *PolizaHandler {
	return &PolizaHandler{validationService: vs}
}

func NewPolizaHandlerWithService(vs validationServicer) *PolizaHandler {
	return &PolizaHandler{validationService: vs}
}

// ValidatePoliza validates a fianza policy
//
//	@Summary		Validate fianza policy
//	@Description	Scrapes the afianzadora website and returns the validation result. Results are cached for CACHE_TTL_HOURS. Use force_refresh=true to bypass cache.
//	@Tags			polizas
//	@Accept			json
//	@Produce		json
//	@Param			request	body		models.ValidationRequest	true	"Validation request"
//	@Success		200		{object}	models.ValidationResponse
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		500		{object}	models.ErrorResponse
//	@Router			/api/v1/polizas/validate [post]
func (h *PolizaHandler) ValidatePoliza(c *fiber.Ctx) error {
	var req models.ValidationRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "invalid request body",
		})
	}

	result, err := h.validationService.Validate(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: err.Error(),
		})
	}

	return c.JSON(result)
}

// ListProviders returns all registered afianzadoras
//
//	@Summary		List supported providers
//	@Description	Returns the keys of all afianzadoras currently registered and available for validation
//	@Tags			polizas
//	@Produce		json
//	@Success		200	{object}	models.ProvidersResponse
//	@Router			/api/v1/polizas/providers [get]
func (h *PolizaHandler) ListProviders(c *fiber.Ctx) error {
	return c.JSON(models.ProvidersResponse{
		Providers: []string{
			"MAPFRE",
			"ASERTA_FIANZAS_SEGUROS",
			"INSURGENTES_AFIANZADORA",
			"BERKLEY_MEXICO_FIANZAS",
			"DORAMA",
			"FIANZAS_Y_CAUCIONES_ATLAS",
			"SOFIMEX",
			"TOKIO_MARINE",
		},
	})
}
