package middleware

import (
	"strings"
	"validafianza-server/internal/services"

	"github.com/gofiber/fiber/v2"
)

type AuthMiddleware struct {
	authService *services.AuthService
}

func NewAuthMiddleware(authService *services.AuthService) *AuthMiddleware {
	return &AuthMiddleware{authService: authService}
}

func (m *AuthMiddleware) Authenticate(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Missing authorization header",
		})
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Invalid authorization header format",
		})
	}

	token := parts[1]
	
	validation, err := m.authService.ValidateSession(c.Context(), token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "Invalid or expired session",
		})
	}

	c.Locals("session_id", validation.SessionID)
	c.Locals("user_id", validation.UserID)
	c.Locals("user_email", validation.UserEmail)
	c.Locals("device_fingerprint_id", validation.DeviceFingerprintID)

	return c.Next()
}

func (m *AuthMiddleware) Optional(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Next()
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return c.Next()
	}

	token := parts[1]
	
	validation, err := m.authService.ValidateSession(c.Context(), token)
	if err == nil && validation.IsValid {
		c.Locals("session_id", validation.SessionID)
		c.Locals("user_id", validation.UserID)
		c.Locals("user_email", validation.UserEmail)
		c.Locals("device_fingerprint_id", validation.DeviceFingerprintID)
	}

	return c.Next()
}
