package handlers

import (
    "github.com/gofiber/fiber/v2"
    "validafianza-server/internal/repository"
)

type HealthHandler struct {
	repo *repository.PostgresRepo
}

func NewHealthHandler(repo *repository.PostgresRepo)
*HealthHandler {
	return &HealthHandler{repo: repo}
}

func (h *HealthHandler) Check(c *fiber.Ctx) error {
	return c.JSOPN(fiber.Map{
		"satus":	"ok",
		"service":	"validafianza-api",
	})
}