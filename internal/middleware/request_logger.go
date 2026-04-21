package middleware

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
)

func RequestLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		status := c.Response().StatusCode()
		latency := time.Since(start)

		attrs := []any{
			"method", c.Method(),
			"path", c.Path(),
			"status", status,
			"latency_ms", latency.Milliseconds(),
			"ip", c.IP(),
			"bytes_out", len(c.Response().Body()),
		}

		if reqID := c.Get("X-Request-ID"); reqID != "" {
			attrs = append(attrs, "request_id", reqID)
		}

		if status >= 500 {
			slog.Error("request", attrs...)
		} else if status >= 400 {
			slog.Warn("request", attrs...)
		} else {
			slog.Info("request", attrs...)
		}

		return err
	}
}
