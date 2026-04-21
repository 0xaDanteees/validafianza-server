package handlers

import (
    "github.com/gofiber/fiber/v2"
    "validafianza-server/internal/models"
    "validafianza-server/internal/services"
)

type PolizaHandler struct {
	validationService *services.ValidationService
}

func NewPolizaHandler(vs *services.ValidationService) *PolizaHandler {
	return &PolizaHandler{validationService: vs}
}

func (h *PolizaHandler) ValidatePoliza( c *fiber.Ctx) error {
	var req models.ValidationRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid req body",
		})
	}

	result, err := h.validationService.Validate(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(result)
}


func (h *PolizaHandler) ListProviders(c *fiber.Ctx) error {
    providers := []string{
        "MAPFRE",
        "ASERTA_FIANZAS_SEGUROS",
        "INSURGENTES_AFIANZADORA",
        "BERKLEY_MEXICO_FIANZAS",
        "DORAMA",
        "FIANZAS_Y_CAUCIONES_ATLAS",
        "SOFIMEX",
        "TOKIO_MARINE",
    }
    
    return c.JSON(fiber.Map{"providers": providers})
}