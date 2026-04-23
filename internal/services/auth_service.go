package services

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
	"time"

	"validafianza-server/internal/models"
	"validafianza-server/internal/repository"

	"github.com/google/uuid"
)

type AuthService struct {
	repo           *repository.AuthRepository
	notificationSvc *NotificationService
}

func NewAuthService(repo *repository.AuthRepository, notificationSvc *NotificationService) *AuthService {
	return &AuthService{
		repo:           repo,
		notificationSvc: notificationSvc,
	}
}

func (s *AuthService) generatePasscode() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	
	code := base32.StdEncoding.EncodeToString(b)
	code = strings.ToUpper(code[:6])
	
	return code, nil
}

func (s *AuthService) generateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base32.StdEncoding.EncodeToString(b), nil
}

func (s *AuthService) RegisterOrLogin(ctx context.Context, email string, phone *string, fullName *string, companyName *string, ipAddress *string, userAgent *string) (*models.Passcode, error) {
	rateLimitKey := fmt.Sprintf("auth:email:%s", email)
	allowed, err := s.repo.CheckRateLimit(ctx, rateLimitKey, 3, 300)
	if err != nil {
		return nil, fmt.Errorf("rate limit check failed: %w", err)
	}
	if !allowed {
		return nil, fmt.Errorf("rate limit exceeded, try again in 5 minutes")
	}

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		user, err = s.repo.CreateUser(ctx, email, phone, fullName, companyName)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	code, err := s.generatePasscode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate passcode: %w", err)
	}

	deliveryMethod := "email"
	deliveryTarget := email
	
	expiresAt := time.Now().Add(10 * time.Minute)
	
	passcode, err := s.repo.CreatePasscode(ctx, user.ID, code, deliveryMethod, deliveryTarget, ipAddress, userAgent, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create passcode: %w", err)
	}

	if err := s.notificationSvc.SendPasscode(ctx, deliveryTarget, code, deliveryMethod); err != nil {
		return nil, fmt.Errorf("failed to send passcode: %w", err)
	}

	return passcode, nil
}

func (s *AuthService) VerifyPasscode(ctx context.Context, code string, fingerprintData string, deviceName *string, ipAddress *string, userAgent *string) (string, *models.User, error) {
	rateLimitKey := fmt.Sprintf("auth:verify:%s", code)
	allowed, err := s.repo.CheckRateLimit(ctx, rateLimitKey, 5, 60)
	if err != nil {
		return "", nil, fmt.Errorf("rate limit check failed: %w", err)
	}
	if !allowed {
		return "", nil, fmt.Errorf("too many verification attempts, try again later")
	}

	passcode, err := s.repo.GetPasscode(ctx, code)
	if err != nil {
		return "", nil, fmt.Errorf("invalid or expired passcode")
	}

	if passcode.Attempts >= passcode.MaxAttempts {
		return "", nil, fmt.Errorf("passcode has exceeded maximum attempts")
	}

	if err := s.repo.IncrementPasscodeAttempts(ctx, passcode.ID); err != nil {
		return "", nil, fmt.Errorf("failed to update passcode: %w", err)
	}

	user, err := s.repo.GetUserByID(ctx, passcode.UserID)
	if err != nil {
		return "", nil, fmt.Errorf("user not found: %w", err)
	}

	deviceAllowed, err := s.repo.CheckDeviceLimit(ctx, user.ID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to check device limit: %w", err)
	}
	if !deviceAllowed {
		return "", nil, fmt.Errorf("maximum device limit reached, please logout from another device")
	}

	device, err := s.repo.GetOrCreateDeviceFingerprint(ctx, user.ID, fingerprintData, deviceName, userAgent, ipAddress)
	if err != nil {
		return "", nil, fmt.Errorf("failed to register device: %w", err)
	}

	token, err := s.generateSessionToken()
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate session token: %w", err)
	}

	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	_, err = s.repo.CreateSession(ctx, user.ID, &device.ID, token, ipAddress, userAgent, expiresAt)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create session: %w", err)
	}

	if err := s.repo.MarkPasscodeAsUsed(ctx, passcode.ID); err != nil {
		return "", nil, fmt.Errorf("failed to mark passcode as used: %w", err)
	}

	return token, user, nil
}

func (s *AuthService) ValidateSession(ctx context.Context, token string) (*models.SessionValidation, error) {
	validation, err := s.repo.ValidateSession(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("session validation failed: %w", err)
	}

	if !validation.IsValid {
		return nil, fmt.Errorf("invalid or expired session")
	}

	return validation, nil
}

func (s *AuthService) Logout(ctx context.Context, sessionID uuid.UUID) error {
	return s.repo.DeleteSession(ctx, sessionID)
}

func (s *AuthService) LogoutAllDevices(ctx context.Context, userID uuid.UUID) error {
	return s.repo.DeleteUserSessions(ctx, userID)
}

func (s *AuthService) CleanupExpiredData(ctx context.Context) error {
	sessions, passcodes, err := s.repo.CleanupExpiredData(ctx)
	if err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}
	
	fmt.Printf("Cleanup completed: %d sessions, %d passcodes deleted\n", sessions, passcodes)
	return nil
}
