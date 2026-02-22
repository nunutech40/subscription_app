package handler

import (
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nununugraha/sains-api/internal/repository"
	"github.com/nununugraha/sains-api/internal/service"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	authService  *service.AuthService
	tokenService *service.TokenService
	queries      *repository.Queries
	refreshDays  int
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(
	authService *service.AuthService,
	tokenService *service.TokenService,
	queries *repository.Queries,
	refreshDays int,
) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		tokenService: tokenService,
		queries:      queries,
		refreshDays:  refreshDays,
	}
}

// ── Request/Response types ──────────────────────────────────────────

type registerRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Name     string `json:"name" binding:"required,min=2"`
	Password string `json:"password" binding:"required,min=8"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type authData struct {
	User        userDTO `json:"user"`
	AccessToken string  `json:"access_token"`
	ExpiresIn   int     `json:"expires_in"` // seconds
}

type userDTO struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

// ── toDTO ───────────────────────────────────────────────────────────

func toUserDTO(u repository.User) userDTO {
	return userDTO{
		ID:    uuidToString(u.ID),
		Email: u.Email,
		Name:  u.Name.String,
		Role:  u.Role.String,
	}
}

// ── Handlers ────────────────────────────────────────────────────────

// Register handles POST /api/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondBadRequest(c, "Data tidak valid: "+err.Error())
		return
	}

	result, err := h.authService.Register(c.Request.Context(), service.RegisterInput{
		Email:    req.Email,
		Name:     req.Name,
		Password: req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmailAlreadyExists):
			RespondConflict(c, "Email sudah terdaftar")
		case errors.Is(err, service.ErrQuotaFull):
			RespondQuotaFull(c, "Kuota subscriber sedang penuh, coba lagi nanti")
		default:
			RespondInternalError(c)
		}
		return
	}

	RespondSuccess(c, http.StatusCreated, toUserDTO(result.User), "Registrasi berhasil! Silakan login.")
}

// Login handles POST /api/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondBadRequest(c, "Data tidak valid: "+err.Error())
		return
	}

	result, err := h.authService.Login(c.Request.Context(), service.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			RespondError(c, http.StatusUnauthorized, ErrCodeInvalidCredentials, "Email atau password salah")
			return
		}
		RespondInternalError(c)
		return
	}

	// Generate access token
	userID := uuidToString(result.User.ID)
	accessToken, err := h.tokenService.GenerateAccessToken(
		userID,
		result.User.Email,
		result.User.Role.String,
	)
	if err != nil {
		RespondInternalError(c)
		return
	}

	// Revoke existing sessions (single session rule)
	_ = h.queries.RevokeAllUserSessions(c.Request.Context(), result.User.ID)

	// Hash refresh token for storage
	refreshHash, err := service.HashRefreshToken(result.RefreshToken)
	if err != nil {
		RespondInternalError(c)
		return
	}

	// Create new session
	_, err = h.queries.CreateSession(c.Request.Context(), repository.CreateSessionParams{
		UserID:           result.User.ID,
		RefreshTokenHash: refreshHash,
		DeviceFingerprint: pgtype.Text{
			String: c.GetHeader("X-Device-Fingerprint"),
			Valid:  c.GetHeader("X-Device-Fingerprint") != "",
		},
		IpAtLogin: parseIP(c.ClientIP()),
		UserAgent: pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().AddDate(0, 0, h.refreshDays),
			Valid: true,
		},
	})
	if err != nil {
		RespondInternalError(c)
		return
	}

	// Set refresh token as httpOnly cookie
	c.SetCookie(
		"refresh_token",
		result.RefreshToken,
		h.refreshDays*24*60*60,
		"/api/auth",
		"",
		true,
		true,
	)

	RespondSuccess(c, http.StatusOK, authData{
		User:        toUserDTO(result.User),
		AccessToken: accessToken,
		ExpiresIn:   3600,
	})
}

// Logout handles POST /api/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		RespondUnauthorized(c, "Tidak terautentikasi")
		return
	}

	userID := stringToUUID(userIDStr.(string))
	_ = h.queries.RevokeAllUserSessions(c.Request.Context(), userID)

	// Clear cookie
	c.SetCookie("refresh_token", "", -1, "/api/auth", "", true, true)

	RespondSuccess(c, http.StatusOK, nil, "Berhasil logout")
}

// Me handles GET /api/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		RespondUnauthorized(c, "Tidak terautentikasi")
		return
	}

	user, err := h.queries.GetUserByID(c.Request.Context(), stringToUUID(userIDStr.(string)))
	if err != nil {
		RespondNotFound(c, "User tidak ditemukan")
		return
	}

	RespondSuccess(c, http.StatusOK, toUserDTO(user))
}

// ── UUID Helpers ────────────────────────────────────────────────────

func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func stringToUUID(s string) pgtype.UUID {
	var u pgtype.UUID
	parsed, err := parseUUID(s)
	if err != nil {
		return u
	}
	u.Bytes = parsed
	u.Valid = true
	return u
}

func parseUUID(s string) ([16]byte, error) {
	var uuid [16]byte
	clean := ""
	for _, c := range s {
		if c != '-' {
			clean += string(c)
		}
	}
	if len(clean) != 32 {
		return uuid, fmt.Errorf("invalid UUID: %s", s)
	}
	for i := 0; i < 16; i++ {
		var b byte
		fmt.Sscanf(clean[i*2:i*2+2], "%02x", &b)
		uuid[i] = b
	}
	return uuid, nil
}

func parseIP(ip string) *netip.Addr {
	if ip == "" {
		return nil
	}
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return nil
	}
	return &addr
}
