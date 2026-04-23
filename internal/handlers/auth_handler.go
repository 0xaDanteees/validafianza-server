package handlers

import (
	"context"
	"validafianza-server/internal/models"
	"validafianza-server/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type authServicer interface {
	RegisterOrLogin(ctx context.Context, email string, phone *string, fullName *string, companyName *string, ipAddress *string, userAgent *string) (*models.Passcode, error)
	VerifyPasscode(ctx context.Context, code string, fingerprintData string, deviceName *string, ipAddress *string, userAgent *string) (string, *models.User, error)
	ValidateSession(ctx context.Context, token string) (*models.SessionValidation, error)
	Logout(ctx context.Context, sessionID uuid.UUID) error
	LogoutAllDevices(ctx context.Context, userID uuid.UUID) error
	CleanupExpiredData(ctx context.Context) error
}

type AuthHandler struct {
	authService authServicer
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func NewAuthHandlerWithService(authService authServicer) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type RegisterLoginRequest struct {
	Email       string  `json:"email" validate:"required,email"`
	Phone       *string `json:"phone,omitempty"`
	FullName    *string `json:"full_name,omitempty"`
	CompanyName *string `json:"company_name,omitempty"`
}

type VerifyPasscodeRequest struct {
	Code            string  `json:"code" validate:"required,min=6,max=8"`
	FingerprintData string  `json:"fingerprint_data" validate:"required"`
	DeviceName      *string `json:"device_name,omitempty"`
}

type AuthResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// RegisterOrLogin initiates passwordless authentication
//
//	@Summary		Register or login with email
//	@Description	Sends a one-time passcode to the user's email. Creates user if doesn't exist.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		RegisterLoginRequest	true	"User email and optional info"
//	@Success		200		{object}	AuthResponse
//	@Failure		400		{object}	AuthResponse
//	@Router			/api/auth/register-or-login [post]
func (h *AuthHandler) RegisterOrLogin(c *fiber.Ctx) error {
	var req RegisterLoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Message: "Invalid request body",
		})
	}

	ipAddress := c.IP()
	userAgent := c.Get("User-Agent")

	passcode, err := h.authService.RegisterOrLogin(
		c.Context(),
		req.Email,
		req.Phone,
		req.FullName,
		req.CompanyName,
		&ipAddress,
		&userAgent,
	)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(AuthResponse{
		Success: true,
		Message: "Passcode sent successfully",
		Data: fiber.Map{
			"delivery_method": passcode.DeliveryMethod,
			"delivery_target": maskEmail(passcode.DeliveryTarget),
			"expires_at":      passcode.ExpiresAt,
		},
	})
}

// VerifyPasscode verifies the OTP code and creates session
//
//	@Summary		Verify passcode
//	@Description	Validates the one-time passcode and returns authentication token
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		VerifyPasscodeRequest	true	"Passcode and device info"
//	@Success		200		{object}	AuthResponse
//	@Failure		400		{object}	AuthResponse
//	@Failure		401		{object}	AuthResponse
//	@Router			/api/auth/verify [post]
func (h *AuthHandler) VerifyPasscode(c *fiber.Ctx) error {
	var req VerifyPasscodeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(AuthResponse{
			Success: false,
			Message: "Invalid request body",
		})
	}

	ipAddress := c.IP()
	userAgent := c.Get("User-Agent")

	token, user, err := h.authService.VerifyPasscode(
		c.Context(),
		req.Code,
		req.FingerprintData,
		req.DeviceName,
		&ipAddress,
		&userAgent,
	)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(AuthResponse{
			Success: false,
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(AuthResponse{
		Success: true,
		Message: "Authentication successful",
		Data: fiber.Map{
			"token": token,
			"user": fiber.Map{
				"id":           user.ID,
				"email":        user.Email,
				"full_name":    user.FullName,
				"company_name": user.CompanyName,
				"is_verified":  user.IsVerified,
			},
		},
	})
}

// GetProfile returns authenticated user profile
//
//	@Summary		Get user profile
//	@Description	Returns the authenticated user's profile information
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	AuthResponse
//	@Failure		401	{object}	AuthResponse
//	@Router			/api/auth/profile [get]
func (h *AuthHandler) GetProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(AuthResponse{
			Success: false,
			Message: "Unauthorized",
		})
	}

	return c.Status(fiber.StatusOK).JSON(AuthResponse{
		Success: true,
		Data: fiber.Map{
			"user_id": userID,
			"email":   c.Locals("user_email"),
		},
	})
}

// Logout terminates the current session
//
//	@Summary		Logout
//	@Description	Terminates the authenticated user's session
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	AuthResponse
//	@Failure		401	{object}	AuthResponse
//	@Router			/api/auth/logout [post]
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	sessionID := c.Locals("session_id")
	if sessionID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(AuthResponse{
			Success: false,
			Message: "No active session",
		})
	}

	sessionUUID, ok := sessionID.(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(AuthResponse{
			Success: false,
			Message: "Invalid session ID",
		})
	}

	err := h.authService.Logout(c.Context(), sessionUUID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(AuthResponse{
			Success: false,
			Message: "Failed to logout: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(AuthResponse{
		Success: true,
		Message: "Logged out successfully",
	})
}

func maskEmail(email string) string {
	if len(email) < 3 {
		return "***"
	}

	parts := []rune(email)
	atIndex := -1
	for i, r := range parts {
		if r == '@' {
			atIndex = i
			break
		}
	}

	if atIndex == -1 {
		return "***"
	}

	visibleStart := 2
	if atIndex < 2 {
		visibleStart = 1
	}

	for i := visibleStart; i < atIndex; i++ {
		parts[i] = '*'
	}

	return string(parts)
}
