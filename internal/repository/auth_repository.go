package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"time"

	"validafianza-server/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthRepository struct {
	db *pgxpool.Pool
}

func NewAuthRepository(db *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) CreateUser(ctx context.Context, email string, phone *string, fullName *string, companyName *string) (*models.User, error) {
	user := &models.User{}
	query := `
		INSERT INTO users (email, phone, full_name, company_name)
		VALUES ($1, $2, $3, $4)
		RETURNING id, email, phone, full_name, company_name, is_active, is_verified, max_devices, created_at, updated_at
	`
	err := r.db.QueryRow(ctx, query, email, phone, fullName, companyName).Scan(
		&user.ID, &user.Email, &user.Phone, &user.FullName, &user.CompanyName,
		&user.IsActive, &user.IsVerified, &user.MaxDevices, &user.CreatedAt, &user.UpdatedAt,
	)
	return user, err
}

func (r *AuthRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, email, phone, full_name, company_name, is_active, is_verified, max_devices, created_at, updated_at
		FROM users
		WHERE email = $1 AND is_active = true
	`
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.Phone, &user.FullName, &user.CompanyName,
		&user.IsActive, &user.IsVerified, &user.MaxDevices, &user.CreatedAt, &user.UpdatedAt,
	)
	return user, err
}

func (r *AuthRepository) GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, email, phone, full_name, company_name, is_active, is_verified, max_devices, created_at, updated_at
		FROM users
		WHERE id = $1 AND is_active = true
	`
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&user.ID, &user.Email, &user.Phone, &user.FullName, &user.CompanyName,
		&user.IsActive, &user.IsVerified, &user.MaxDevices, &user.CreatedAt, &user.UpdatedAt,
	)
	return user, err
}

func (r *AuthRepository) CreatePasscode(ctx context.Context, userID uuid.UUID, code string, deliveryMethod string, deliveryTarget string, ipAddress *string, userAgent *string, expiresAt time.Time) (*models.Passcode, error) {
	passcode := &models.Passcode{}

	var ipNet *net.IP
	if ipAddress != nil && *ipAddress != "" {
		ip := net.ParseIP(*ipAddress)
		if ip != nil {
			ipNet = &ip
		}
	}

	query := `
		INSERT INTO passcodes (user_id, code, delivery_method, delivery_target, ip_address, user_agent, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, code, delivery_method, delivery_target, is_used, attempts, max_attempts, ip_address, user_agent, expires_at, created_at
	`
	err := r.db.QueryRow(ctx, query, userID, code, deliveryMethod, deliveryTarget, ipNet, userAgent, expiresAt).Scan(
		&passcode.ID, &passcode.UserID, &passcode.Code, &passcode.DeliveryMethod, &passcode.DeliveryTarget,
		&passcode.IsUsed, &passcode.Attempts, &passcode.MaxAttempts, &passcode.IPAddress, &passcode.UserAgent,
		&passcode.ExpiresAt, &passcode.CreatedAt,
	)
	return passcode, err
}

func (r *AuthRepository) GetPasscode(ctx context.Context, code string) (*models.Passcode, error) {
	passcode := &models.Passcode{}
	query := `
		SELECT id, user_id, code, delivery_method, delivery_target, is_used, attempts, max_attempts, ip_address, user_agent, expires_at, used_at, created_at
		FROM passcodes
		WHERE code = $1 AND is_used = false AND expires_at > NOW()
	`
	err := r.db.QueryRow(ctx, query, code).Scan(
		&passcode.ID, &passcode.UserID, &passcode.Code, &passcode.DeliveryMethod, &passcode.DeliveryTarget,
		&passcode.IsUsed, &passcode.Attempts, &passcode.MaxAttempts, &passcode.IPAddress, &passcode.UserAgent,
		&passcode.ExpiresAt, &passcode.UsedAt, &passcode.CreatedAt,
	)
	return passcode, err
}

func (r *AuthRepository) IncrementPasscodeAttempts(ctx context.Context, passcodeID uuid.UUID) error {
	query := `UPDATE passcodes SET attempts = attempts + 1 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, passcodeID)
	return err
}

func (r *AuthRepository) MarkPasscodeAsUsed(ctx context.Context, passcodeID uuid.UUID) error {
	query := `UPDATE passcodes SET is_used = true, used_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, passcodeID)
	return err
}

func (r *AuthRepository) GetOrCreateDeviceFingerprint(ctx context.Context, userID uuid.UUID, fingerprintData string, deviceName *string, userAgent *string, ipAddress *string) (*models.DeviceFingerprint, error) {
	hash := sha256.Sum256([]byte(fingerprintData))
	fingerprintHash := hex.EncodeToString(hash[:])

	var ipNet *net.IP
	if ipAddress != nil && *ipAddress != "" {
		ip := net.ParseIP(*ipAddress)
		if ip != nil {
			ipNet = &ip
		}
	}

	device := &models.DeviceFingerprint{}

	query := `
		INSERT INTO device_fingerprints (user_id, fingerprint_hash, device_name, user_agent, ip_address)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (fingerprint_hash) DO UPDATE
		SET last_seen_at = NOW(), user_agent = EXCLUDED.user_agent, ip_address = EXCLUDED.ip_address
		RETURNING id, user_id, fingerprint_hash, device_name, user_agent, ip_address, is_trusted, last_seen_at, created_at
	`

	err := r.db.QueryRow(ctx, query, userID, fingerprintHash, deviceName, userAgent, ipNet).Scan(
		&device.ID, &device.UserID, &device.FingerprintHash, &device.DeviceName, &device.UserAgent,
		&device.IPAddress, &device.IsTrusted, &device.LastSeenAt, &device.CreatedAt,
	)
	return device, err
}

func (r *AuthRepository) CheckDeviceLimit(ctx context.Context, userID uuid.UUID) (bool, error) {
	var allowed bool
	query := `SELECT check_device_limit($1)`
	err := r.db.QueryRow(ctx, query, userID).Scan(&allowed)
	return allowed, err
}

func (r *AuthRepository) CreateSession(ctx context.Context, userID uuid.UUID, deviceFingerprintID *uuid.UUID, token string, ipAddress *string, userAgent *string, expiresAt time.Time) (*models.Session, error) {
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	var ipNet *net.IP
	if ipAddress != nil && *ipAddress != "" {
		ip := net.ParseIP(*ipAddress)
		if ip != nil {
			ipNet = &ip
		}
	}

	session := &models.Session{}
	query := `
		INSERT INTO sessions (user_id, device_fingerprint_id, token_hash, ip_address, user_agent, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, device_fingerprint_id, token_hash, ip_address, user_agent, last_activity_at, expires_at, created_at
	`
	err := r.db.QueryRow(ctx, query, userID, deviceFingerprintID, tokenHash, ipNet, userAgent, expiresAt).Scan(
		&session.ID, &session.UserID, &session.DeviceFingerprintID, &session.TokenHash,
		&session.IPAddress, &session.UserAgent, &session.LastActivityAt, &session.ExpiresAt, &session.CreatedAt,
	)
	return session, err
}

func (r *AuthRepository) ValidateSession(ctx context.Context, token string) (*models.SessionValidation, error) {
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	validation := &models.SessionValidation{}
	query := `SELECT * FROM validate_session($1)`
	err := r.db.QueryRow(ctx, query, tokenHash).Scan(
		&validation.SessionID, &validation.UserID, &validation.UserEmail,
		&validation.DeviceFingerprintID, &validation.IsValid,
	)
	return validation, err
}

func (r *AuthRepository) DeleteSession(ctx context.Context, sessionID uuid.UUID) error {
	query := `DELETE FROM sessions WHERE id = $1`
	_, err := r.db.Exec(ctx, query, sessionID)
	return err
}

func (r *AuthRepository) DeleteUserSessions(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM sessions WHERE user_id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}

func (r *AuthRepository) CleanupExpiredData(ctx context.Context) (int, int, error) {
	var deletedSessions, deletedPasscodes int
	query := `SELECT * FROM cleanup_expired_data()`
	err := r.db.QueryRow(ctx, query).Scan(&deletedSessions, &deletedPasscodes)
	return deletedSessions, deletedPasscodes, err
}

func (r *AuthRepository) CheckRateLimit(ctx context.Context, key string, limit int, windowSeconds int) (bool, error) {
	var allowed bool
	query := `SELECT check_rate_limit($1, $2, $3)`
	err := r.db.QueryRow(ctx, query, key, limit, windowSeconds).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("rate limit check failed: %w", err)
	}
	return allowed, nil
}
