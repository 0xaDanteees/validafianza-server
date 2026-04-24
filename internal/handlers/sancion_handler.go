package handlers

import (
	"context"

	"validafianza-server/internal/models"
	"validafianza-server/internal/services"

	"github.com/gofiber/fiber/v2"
)

type SancionServiceInterface interface {
	ValidarRFCSCG(ctx context.Context, rfc string) (*models.SancionResultSCG, error)
	ValidarRFCSFP(ctx context.Context, rfc string) (*models.SancionResultSFP, error)
}

type SancionHandler struct {
	service SancionServiceInterface
}

func NewSancionHandler(service *services.SancionService) *SancionHandler {
	return &SancionHandler{service: service}
}

// ValidarRFCRequest is the payload for sanction validation endpoints
type ValidarRFCRequest struct {
	RFC string `json:"rfc" validate:"required" example:"SIR130227V77"`
}

// SancionResponse is the response wrapper for sanction validation
type SancionResponse struct {
	Success bool        `json:"success" example:"true"`
	Message string      `json:"message,omitempty" example:"RFC validated successfully"`
	Data    interface{} `json:"data,omitempty"`
}

// ValidarRFCSCG validates if an RFC is sanctioned by SCG
//
//	@Summary		Validate RFC against SCG sanctions
//	@Description	Checks if an RFC is sanctioned by SCG (CDMX Comptroller). Results are cached for CACHE_TTL_HOURS.
//	@Tags			sanciones
//	@Accept			json
//	@Produce		json
//	@Param			request	body		ValidarRFCRequest	true	"RFC to validate"
//	@Success		200		{object}	SancionResponse
//	@Failure		400		{object}	SancionResponse
//	@Failure		500		{object}	SancionResponse
//	@Security		BearerAuth
//	@Router			/api/v1/sanciones/scg [post]
func (h *SancionHandler) ValidarRFCSCG(c *fiber.Ctx) error {
	var req ValidarRFCRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(SancionResponse{
			Success: false,
			Message: "Invalid request body",
		})
	}

	if req.RFC == "" {
		return c.Status(fiber.StatusBadRequest).JSON(SancionResponse{
			Success: false,
			Message: "RFC is required",
		})
	}

	result, err := h.service.ValidarRFCSCG(c.Context(), req.RFC)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(SancionResponse{
			Success: false,
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(SancionResponse{
		Success: true,
		Data:    result,
	})
}

// ValidarRFCSFP validates if an RFC is sanctioned by SFP
//
//	@Summary		Validate RFC against SFP sanctions
//	@Description	Checks if an RFC is sanctioned by SFP (Public Function). Results are cached for CACHE_TTL_HOURS. Note: This endpoint uses browser automation and may take 60-90 seconds.
//	@Tags			sanciones
//	@Accept			json
//	@Produce		json
//	@Param			request	body		ValidarRFCRequest	true	"RFC to validate"
//	@Success		200		{object}	SancionResponse
//	@Failure		400		{object}	SancionResponse
//	@Failure		500		{object}	SancionResponse
//	@Security		BearerAuth
//	@Router			/api/v1/sanciones/sfp [post]
func (h *SancionHandler) ValidarRFCSFP(c *fiber.Ctx) error {
	var req ValidarRFCRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(SancionResponse{
			Success: false,
			Message: "Invalid request body",
		})
	}

	if req.RFC == "" {
		return c.Status(fiber.StatusBadRequest).JSON(SancionResponse{
			Success: false,
			Message: "RFC is required",
		})
	}

	result, err := h.service.ValidarRFCSFP(c.Context(), req.RFC)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(SancionResponse{
			Success: false,
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(SancionResponse{
		Success: true,
		Data:    result,
	})
}
