package handler

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/netip"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nununugraha/sains-api/internal/repository"
	"github.com/nununugraha/sains-api/internal/service"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	authService    *service.AuthService
	tokenService   *service.TokenService
	anomalyService *service.AnomalyService
	emailService   *service.EmailService
	queries        *repository.Queries
	refreshDays    int
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(
	authService *service.AuthService,
	tokenService *service.TokenService,
	anomalyService *service.AnomalyService,
	emailService *service.EmailService,
	queries *repository.Queries,
	refreshDays int,
) *AuthHandler {
	return &AuthHandler{
		authService:    authService,
		tokenService:   tokenService,
		anomalyService: anomalyService,
		emailService:   emailService,
		queries:        queries,
		refreshDays:    refreshDays,
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

type guestLoginRequest struct {
	Code           string `json:"code" binding:"required"`
	Email          string `json:"email" binding:"required,email"`
	ReferralSource string `json:"referral_source"`
}

type guestVerifyRequest struct {
	Code           string `json:"code" binding:"required"`
	Email          string `json:"email" binding:"required,email"`
	OTP            string `json:"otp" binding:"required,len=6"`
	ReferralSource string `json:"referral_source"`
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
		log.Printf("login error: %v", err)
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

	// Check anomaly before revoking (needs old session data)
	go h.anomalyService.CheckLoginAnomaly(context.Background(), result.User.ID, c.ClientIP(), c.GetHeader("User-Agent"))

	// Revoke existing sessions (single session rule)
	_ = h.queries.RevokeAllUserSessions(c.Request.Context(), result.User.ID)

	// Hash refresh token for storage
	refreshHash, err := service.HashRefreshToken(result.RefreshToken)
	if err != nil {
		log.Printf("hash refresh token error: %v", err)
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
		log.Printf("create session error: %v", err)
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

// GuestLogin handles POST /api/auth/guest-login
// Step 1: Validate code + email → send OTP to email → return pending
func (h *AuthHandler) GuestLogin(c *gin.Context) {
	var req guestLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondBadRequest(c, "Data tidak valid: "+err.Error())
		return
	}

	ctx := c.Request.Context()

	// 1. Validate guest code
	guestCode, err := h.queries.GetGuestCodeByCode(ctx, req.Code)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, ErrCodeInvalidCredentials, "Kode guest tidak valid atau sudah tidak aktif")
		return
	}

	// Check expiry
	if guestCode.ExpiresAt.Time.Before(time.Now()) {
		RespondError(c, http.StatusUnauthorized, ErrCodeInvalidCredentials, "Kode guest sudah expired")
		return
	}

	// 2. Check guest quota
	activeGuests, _ := h.queries.CountActiveGuestSessions(ctx)
	maxGuestsStr := "50"
	if cfg, cfgErr := h.queries.GetConfig(ctx, "max_active_guests"); cfgErr == nil {
		maxGuestsStr = cfg.Value
	}
	maxGuests := 50
	fmt.Sscanf(maxGuestsStr, "%d", &maxGuests)

	if int(activeGuests) >= maxGuests {
		RespondQuotaFull(c, "Kuota guest sedang penuh, coba lagi nanti")
		return
	}

	// 3. Check login count per email
	existingLogin, loginErr := h.queries.GetGuestLogin(ctx, repository.GetGuestLoginParams{
		GuestCodeID: guestCode.ID,
		Email:       req.Email,
	})
	if loginErr == nil && existingLogin.LoginCount.Int32 >= guestCode.MaxLoginsPerEmail.Int32 {
		RespondError(c, http.StatusForbidden, ErrCodeForbidden,
			fmt.Sprintf("Trial habis. Kamu sudah login %d kali dengan kode ini.", existingLogin.LoginCount.Int32))
		return
	}

	// 4. Rate limit OTP requests (max 3 per 10 min per email)
	recentCount, _ := h.queries.CountRecentOTPs(ctx, req.Email)
	if recentCount >= 3 {
		RespondError(c, http.StatusTooManyRequests, "RATE_LIMITED",
			"Terlalu banyak permintaan OTP. Coba lagi dalam beberapa menit.")
		return
	}

	// 5. Generate 6-digit OTP
	otpCode := generateOTP()

	// 6. Store OTP in DB (expires in 5 minutes)
	_, err = h.queries.CreateGuestOTP(ctx, repository.CreateGuestOTPParams{
		Email:       req.Email,
		GuestCodeID: guestCode.ID,
		OtpCode:     otpCode,
		ExpiresAt:   pgtype.Timestamptz{Time: time.Now().Add(5 * time.Minute), Valid: true},
		Ip:          parseIP(c.ClientIP()),
	})
	if err != nil {
		log.Printf("create OTP error: %v", err)
		RespondInternalError(c)
		return
	}

	// 7. Send OTP email
	go func() {
		emailErr := h.emailService.SendGuestOTPEmail(context.Background(), req.Email, otpCode)
		if emailErr != nil {
			log.Printf("send OTP email error: %v", emailErr)
		}
	}()

	// 8. Cleanup expired OTPs in background
	go func() { _ = h.queries.CleanExpiredOTPs(context.Background()) }()

	RespondSuccess(c, http.StatusOK, gin.H{
		"pending_verification": true,
		"email":                maskEmail(req.Email),
		"expires_in":           300, // 5 minutes
	}, "Kode OTP sudah dikirim ke email kamu")
}

// GuestVerify handles POST /api/auth/guest-verify
// Step 2: Validate OTP → create session → return access token
func (h *AuthHandler) GuestVerify(c *gin.Context) {
	var req guestVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondBadRequest(c, "Data tidak valid: "+err.Error())
		return
	}

	ctx := c.Request.Context()

	// 1. Re-validate guest code (it could have expired between steps)
	guestCode, err := h.queries.GetGuestCodeByCode(ctx, req.Code)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, ErrCodeInvalidCredentials, "Kode guest tidak valid")
		return
	}

	// 2. Validate OTP
	otpRecord, err := h.queries.GetPendingOTP(ctx, repository.GetPendingOTPParams{
		Email:       req.Email,
		GuestCodeID: guestCode.ID,
		OtpCode:     req.OTP,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			RespondError(c, http.StatusUnauthorized, "OTP_INVALID",
				"Kode OTP salah atau sudah expired. Coba minta ulang.")
			return
		}
		RespondInternalError(c)
		return
	}

	// 3. Mark OTP as verified
	_ = h.queries.MarkOTPVerified(ctx, otpRecord.ID)

	// 4. Upsert guest login (increment count + track IP) — only after verified
	clientIP := parseIP(c.ClientIP())
	_, _ = h.queries.UpsertGuestLogin(ctx, repository.UpsertGuestLoginParams{
		GuestCodeID:    guestCode.ID,
		Email:          req.Email,
		ReferralSource: pgtype.Text{String: req.ReferralSource, Valid: req.ReferralSource != ""},
		IpAddress:      clientIP,
	})

	// 5. Generate access token (guest role)
	accessToken, err := h.tokenService.GenerateAccessToken(
		"guest:"+uuidToString(guestCode.ID),
		req.Email,
		"guest",
	)
	if err != nil {
		RespondInternalError(c)
		return
	}

	// 6. Create guest session (24 hours)
	refreshToken := service.GenerateRefreshToken()
	refreshHash, _ := service.HashRefreshToken(refreshToken)

	_, _ = h.queries.CreateSession(ctx, repository.CreateSessionParams{
		GuestCodeID:      guestCode.ID,
		GuestEmail:       pgtype.Text{String: req.Email, Valid: true},
		RefreshTokenHash: refreshHash,
		IpAtLogin:        parseIP(c.ClientIP()),
		UserAgent:        pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(24 * time.Hour),
			Valid: true,
		},
	})

	// Set cookie
	c.SetCookie("refresh_token", refreshToken, 24*60*60, "/api/auth", "", true, true)

	RespondSuccess(c, http.StatusOK, gin.H{
		"access_token": accessToken,
		"expires_in":   3600,
		"role":         "guest",
		"product":      guestCode.ProductID.String,
	}, "Login sebagai guest berhasil")
}

// ── OTP Helpers ─────────────────────────────────────────────────────

func generateOTP() string {
	const digits = "0123456789"
	otp := make([]byte, 6)
	for i := range otp {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			// Fallback to simple approach
			otp[i] = digits[i]
			continue
		}
		otp[i] = digits[n.Int64()]
	}
	return string(otp)
}

func maskEmail(email string) string {
	at := -1
	for i, c := range email {
		if c == '@' {
			at = i
			break
		}
	}
	if at <= 0 {
		return email
	}
	local := email[:at]
	domain := email[at:]
	if len(local) <= 2 {
		return local[:1] + "***" + domain
	}
	return local[:2] + "***" + domain
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
