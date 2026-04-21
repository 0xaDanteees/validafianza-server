package middleware

import (
	"context"
	"log/slog"
	"time"

	"validafianza-server/internal/models"

	"github.com/gofiber/fiber/v2"
)

type rateLimiter interface {
	Allow(ctx context.Context, key string, limit, windowSeconds int) (bool, error)
}

func RateLimit(repo rateLimiter, limit, windowSeconds int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := "ip:" + c.IP()

		ctx, cancel := context.WithTimeout(c.Context(), 300*time.Millisecond)
		defer cancel()

		allowed, err := repo.Allow(ctx, key, limit, windowSeconds)
		if err != nil {
			slog.Warn("rate limit check failed, allowing request",
				"key", key,
				"error", err,
			)
			return c.Next()
		}

		if !allowed {
			slog.Info("rate limit exceeded",
				"key", key,
				"limit", limit,
				"window_seconds", windowSeconds,
			)
			return c.Status(fiber.StatusTooManyRequests).JSON(models.ErrorResponse{
				Error: "rate limit exceeded — slow down",
			})
		}

		return c.Next()
	}
}
