package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nununugraha/sains-api/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailAlreadyExists = errors.New("email sudah terdaftar")
	ErrInvalidCredentials = errors.New("email atau password salah")
	ErrAccountInactive    = errors.New("akun belum aktif")
	ErrQuotaFull          = errors.New("kuota subscriber penuh, coba lagi nanti")
)

// AuthService handles authentication business logic.
type AuthService struct {
	queries *repository.Queries
}

// NewAuthService creates a new AuthService.
func NewAuthService(queries *repository.Queries) *AuthService {
	return &AuthService{queries: queries}
}

// RegisterInput holds registration request data.
type RegisterInput struct {
	Email    string
	Name     string
	Password string
}

// RegisterResult holds registration response data.
type RegisterResult struct {
	User repository.User
}

// Register creates a new user account.
func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*RegisterResult, error) {
	// Check subscriber quota
	quotaFull, err := s.isSubscriberQuotaFull(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check quota: %w", err)
	}
	if quotaFull {
		return nil, ErrQuotaFull
	}

	// Check if email already exists
	_, err = s.queries.GetUserByEmail(ctx, input.Email)
	if err == nil {
		return nil, ErrEmailAlreadyExists
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user, err := s.queries.CreateUser(ctx, repository.CreateUserParams{
		Email:        input.Email,
		Name:         pgtype.Text{String: input.Name, Valid: input.Name != ""},
		PasswordHash: string(hash),
		Role:         pgtype.Text{String: "subscriber", Valid: true},
		IsActive:     pgtype.Bool{Bool: false, Valid: true}, // aktif setelah bayar
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &RegisterResult{User: user}, nil
}

// LoginInput holds login request data.
type LoginInput struct {
	Email    string
	Password string
}

// LoginResult holds login response data.
type LoginResult struct {
	User         repository.User
	RefreshToken string // raw token (will be hashed for storage)
}

// Login authenticates a user with email and password.
func (s *AuthService) Login(ctx context.Context, input LoginInput) (*LoginResult, error) {
	// Find user
	user, err := s.queries.GetUserByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Compare password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Generate refresh token
	refreshToken, err := generateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &LoginResult{
		User:         user,
		RefreshToken: refreshToken,
	}, nil
}

// isSubscriberQuotaFull checks if subscriber capacity has been reached.
func (s *AuthService) isSubscriberQuotaFull(ctx context.Context) (bool, error) {
	// Get max_subscribers from system_config
	cfg, err := s.queries.GetConfig(ctx, "max_subscribers")
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil // no config = no limit
		}
		return false, err
	}

	var maxSubs int
	fmt.Sscanf(cfg.Value, "%d", &maxSubs)
	if maxSubs == 0 {
		return false, nil
	}

	// Count current subscribers
	count, err := s.queries.CountActiveSubscribers(ctx)
	if err != nil {
		return false, err
	}

	return count >= int64(maxSubs), nil
}

// generateRefreshToken creates a cryptographically secure random token.
func generateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// HashRefreshToken hashes a refresh token for storage.
func HashRefreshToken(token string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(token), 10)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyRefreshToken compares a raw token against its hash.
func VerifyRefreshToken(token, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(token)) == nil
}

// SessionInput holds data needed to create a session.
type SessionInput struct {
	UserID            pgtype.UUID
	GuestCodeID       pgtype.UUID
	GuestEmail        string
	RefreshToken      string
	DeviceFingerprint string
	IP                string
	UserAgent         string
	CountryCode       string
	ExpiresAt         time.Time
}
