package middleware

import (
	"fmt"
	"validafianza-server/internal/repository"

	"github.com/gofiber/fiber/v2"
)

type RateLimitMiddleware struct {
	repo *repository.AuthRepository
}

func NewRateLimitMiddleware(repo *repository.AuthRepository) *RateLimitMiddleware {
	return &RateLimitMiddleware{repo: repo}
}

func (m *RateLimitMiddleware) LimitByIP(limit int, windowSeconds int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ip := c.IP()
		key := fmt.Sprintf("ratelimit:ip:%s:%s", c.Path(), ip)
		
		allowed, err := m.repo.CheckRateLimit(c.Context(), key, limit, windowSeconds)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Rate limit check failed",
			})
		}
		
		if !allowed {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"message": "Rate limit exceeded, please try again later",
			})
		}
		
		return c.Next()
	}
}

func (m *RateLimitMiddleware) LimitByUser(limit int, windowSeconds int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("user_id")
		if userID == nil {
			return c.Next()
		}
		
		key := fmt.Sprintf("ratelimit:user:%s:%v", c.Path(), userID)
		
		allowed, err := m.repo.CheckRateLimit(c.Context(), key, limit, windowSeconds)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Rate limit check failed",
			})
		}
		
		if !allowed {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"message": "Rate limit exceeded, please try again later",
			})
		}
		
		return c.Next()
	}
}
