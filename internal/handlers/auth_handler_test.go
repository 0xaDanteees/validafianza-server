package handlers_test

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"validafianza-server/internal/handlers"
	"validafianza-server/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Fake auth service
// ---------------------------------------------------------------------------

type fakeAuthService struct {
	passcode    *models.Passcode
	token       string
	user        *models.User
	validation  *models.SessionValidation
	registerErr error
	verifyErr   error
	validateErr error
	logoutErr   error
}

func (f *fakeAuthService) RegisterOrLogin(_ context.Context, email string, phone *string, fullName *string, companyName *string, ipAddress *string, userAgent *string) (*models.Passcode, error) {
	if f.registerErr != nil {
		return nil, f.registerErr
	}
	return f.passcode, nil
}

func (f *fakeAuthService) VerifyPasscode(_ context.Context, code string, fingerprintData string, deviceName *string, ipAddress *string, userAgent *string) (string, *models.User, error) {
	if f.verifyErr != nil {
		return "", nil, f.verifyErr
	}
	return f.token, f.user, nil
}

func (f *fakeAuthService) ValidateSession(_ context.Context, token string) (*models.SessionValidation, error) {
	if f.validateErr != nil {
		return nil, f.validateErr
	}
	return f.validation, nil
}

func (f *fakeAuthService) Logout(_ context.Context, sessionID uuid.UUID) error {
	return f.logoutErr
}

func (f *fakeAuthService) LogoutAllDevices(_ context.Context, userID uuid.UUID) error {
	return nil
}

func (f *fakeAuthService) CleanupExpiredData(_ context.Context) error {
	return nil
}

// ---------------------------------------------------------------------------
// Test app factory
// ---------------------------------------------------------------------------

func newAuthTestApp(svc *fakeAuthService) *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: fiber.DefaultErrorHandler})
	h := handlers.NewAuthHandlerWithService(svc)

	app.Post("/api/auth/register-or-login", h.RegisterOrLogin)
	app.Post("/api/auth/verify", h.VerifyPasscode)
	app.Get("/api/auth/profile", func(c *fiber.Ctx) error {
		c.Locals("user_id", uuid.New())
		c.Locals("user_email", "test@example.com")
		return h.GetProfile(c)
	})
	app.Post("/api/auth/logout", func(c *fiber.Ctx) error {
		c.Locals("session_id", uuid.New())
		return h.Logout(c)
	})

	return app
}

// ---------------------------------------------------------------------------
// Tests: RegisterOrLogin
// ---------------------------------------------------------------------------

func TestRegisterOrLogin_InvalidJSON_Returns400(t *testing.T) {
	svc := &fakeAuthService{}
	app := newAuthTestApp(svc)

	req := httptest.NewRequest("POST", "/api/auth/register-or-login",
		strings.NewReader(`{not valid json`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "success") {
		t.Errorf("expected success field in body, got: %s", body)
	}
}

func TestRegisterOrLogin_ServiceError_Returns400(t *testing.T) {
	svc := &fakeAuthService{
		registerErr: errors.New("rate limit exceeded"),
	}
	app := newAuthTestApp(svc)

	req := httptest.NewRequest("POST", "/api/auth/register-or-login",
		strings.NewReader(`{"email":"test@example.com"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "rate limit exceeded") {
		t.Errorf("expected error message in body, got: %s", body)
	}
}

func TestRegisterOrLogin_ValidRequest_Returns200(t *testing.T) {
	expiresAt := time.Now().Add(10 * time.Minute)
	svc := &fakeAuthService{
		passcode: &models.Passcode{
			ID:             uuid.New(),
			UserID:         uuid.New(),
			DeliveryMethod: "email",
			DeliveryTarget: "test@example.com",
			ExpiresAt:      expiresAt,
		},
	}
	app := newAuthTestApp(svc)

	req := httptest.NewRequest("POST", "/api/auth/register-or-login",
		strings.NewReader(`{"email":"test@example.com","full_name":"Test User"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "success") {
		t.Errorf("expected success field in body, got: %s", body)
	}
	if !strings.Contains(string(body), "delivery_method") {
		t.Errorf("expected delivery_method field in body, got: %s", body)
	}
}

// ---------------------------------------------------------------------------
// Tests: VerifyPasscode
// ---------------------------------------------------------------------------

func TestVerifyPasscode_InvalidJSON_Returns400(t *testing.T) {
	svc := &fakeAuthService{}
	app := newAuthTestApp(svc)

	req := httptest.NewRequest("POST", "/api/auth/verify",
		strings.NewReader(`{not valid json`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestVerifyPasscode_InvalidCode_Returns401(t *testing.T) {
	svc := &fakeAuthService{
		verifyErr: errors.New("invalid or expired passcode"),
	}
	app := newAuthTestApp(svc)

	req := httptest.NewRequest("POST", "/api/auth/verify",
		strings.NewReader(`{"code":"INVALID","fingerprint_data":"test_hash"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 401 {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "invalid or expired passcode") {
		t.Errorf("expected error message in body, got: %s", body)
	}
}

func TestVerifyPasscode_ValidCode_Returns200WithToken(t *testing.T) {
	email := "test@example.com"
	svc := &fakeAuthService{
		token: "test_token_abc123",
		user: &models.User{
			ID:         uuid.New(),
			Email:      email,
			IsActive:   true,
			IsVerified: true,
		},
	}
	app := newAuthTestApp(svc)

	req := httptest.NewRequest("POST", "/api/auth/verify",
		strings.NewReader(`{"code":"ABC123","fingerprint_data":"test_hash"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "token") {
		t.Errorf("expected token field in body, got: %s", body)
	}
	if !strings.Contains(string(body), email) {
		t.Errorf("expected email in body, got: %s", body)
	}
}

// ---------------------------------------------------------------------------
// Tests: GetProfile
// ---------------------------------------------------------------------------

func TestGetProfile_Authenticated_Returns200(t *testing.T) {
	svc := &fakeAuthService{}
	app := newAuthTestApp(svc)

	req := httptest.NewRequest("GET", "/api/auth/profile", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "user_id") {
		t.Errorf("expected user_id field in body, got: %s", body)
	}
}

// ---------------------------------------------------------------------------
// Tests: Logout
// ---------------------------------------------------------------------------

func TestLogout_Authenticated_Returns200(t *testing.T) {
	svc := &fakeAuthService{}
	app := newAuthTestApp(svc)

	req := httptest.NewRequest("POST", "/api/auth/logout", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "success") {
		t.Errorf("expected success field in body, got: %s", body)
	}
}

func TestLogout_ServiceError_Returns500(t *testing.T) {
	svc := &fakeAuthService{
		logoutErr: errors.New("database error"),
	}
	app := newAuthTestApp(svc)

	req := httptest.NewRequest("POST", "/api/auth/logout", nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("status = %d, want 500", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "database error") {
		t.Errorf("expected error message in body, got: %s", body)
	}
}
