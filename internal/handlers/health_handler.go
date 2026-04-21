package handlers

import (
	"github.com/gofiber/fiber/v2"
	"validafianza-server/internal/repository"
)

type HealthHandler struct {
	repo *repository.PostgresRepo
}

func NewHealthHandler(repo *repository.PostgresRepo) *HealthHandler {
	return &HealthHandler{repo: repo}
}

func (h *HealthHandler) Check(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "ok",
		"service": "validafianza-api",
	})
}
