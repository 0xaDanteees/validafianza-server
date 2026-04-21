package handlers

import (
	"github.com/gofiber/fiber/v2"
	"validafianza-server/internal/models"
	"validafianza-server/internal/repository"
)

type HealthHandler struct {
	repo *repository.PostgresRepo
}

func NewHealthHandler(repo *repository.PostgresRepo) *HealthHandler {
	return &HealthHandler{repo: repo}
}

// Check returns API health status
//
//	@Summary		Health check
//	@Description	Returns service liveness. Does not check DB — use for load balancer probes.
//	@Tags			system
//	@Produce		json
//	@Success		200	{object}	models.HealthResponse
//	@Router			/health [get]
func (h *HealthHandler) Check(c *fiber.Ctx) error {
	return c.JSON(models.HealthResponse{
		Status:  "ok",
		Service: "validafianza-api",
	})
}
