package handler

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nununugraha/sains-api/internal/repository"
)

// GuestHandler handles guest code endpoints.
type GuestHandler struct {
	queries *repository.Queries
}

// NewGuestHandler creates a new GuestHandler.
func NewGuestHandler(queries *repository.Queries) *GuestHandler {
	return &GuestHandler{queries: queries}
}

// ── DTOs ────────────────────────────────────────────────────────────

type guestCodeDTO struct {
	ID                string `json:"id"`
	Code              string `json:"code"`
	ProductID         string `json:"product_id"`
	Label             string `json:"label"`
	MaxLoginsPerEmail int    `json:"max_logins_per_email"`
	IsActive          bool   `json:"is_active"`
	ExpiresAt         string `json:"expires_at"`
	CreatedAt         string `json:"created_at"`
}

type guestLoginDTO struct {
	Email       string `json:"email"`
	LoginCount  int    `json:"login_count"`
	LastLoginAt string `json:"last_login_at"`
}

func toGuestCodeDTO(g repository.GuestCode) guestCodeDTO {
	return guestCodeDTO{
		ID:                uuidToString(g.ID),
		Code:              g.Code,
		ProductID:         g.ProductID.String,
		Label:             g.Label.String,
		MaxLoginsPerEmail: int(g.MaxLoginsPerEmail.Int32),
		IsActive:          g.IsActive.Bool,
		ExpiresAt:         g.ExpiresAt.Time.Format("2006-01-02T15:04:05Z"),
		CreatedAt:         g.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
	}
}

func toGuestLoginDTO(g repository.GuestLogin) guestLoginDTO {
	return guestLoginDTO{
		Email:       g.Email,
		LoginCount:  int(g.LoginCount.Int32),
		LastLoginAt: g.LastLoginAt.Time.Format("2006-01-02T15:04:05Z"),
	}
}

// ── Admin: Generate Guest Code ──────────────────────────────────────

// CreateGuestCode handles POST /api/admin/guest-codes
func (h *GuestHandler) CreateGuestCode(c *gin.Context) {
	var req struct {
		ProductID         string `json:"product_id"`
		Label             string `json:"label" binding:"required"`
		MaxLoginsPerEmail int    `json:"max_logins_per_email"`
		ExpiresHours      int    `json:"expires_hours" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondBadRequest(c, "Data tidak valid: "+err.Error())
		return
	}

	if req.ProductID == "" {
		req.ProductID = "atomic"
	}
	if req.MaxLoginsPerEmail <= 0 {
		req.MaxLoginsPerEmail = 2
	}

	// Generate unique code: ATOM-XXXX
	code := generateGuestCode()

	// Get admin user ID from context
	adminIDStr, _ := c.Get("user_id")
	adminID := stringToUUID(adminIDStr.(string))

	guestCode, err := h.queries.CreateGuestCode(c.Request.Context(), repository.CreateGuestCodeParams{
		Code:              code,
		ProductID:         pgtype.Text{String: req.ProductID, Valid: true},
		Label:             pgtype.Text{String: req.Label, Valid: true},
		MaxLoginsPerEmail: pgtype.Int4{Int32: int32(req.MaxLoginsPerEmail), Valid: true},
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(time.Duration(req.ExpiresHours) * time.Hour),
			Valid: true,
		},
		GeneratedBy: adminID,
	})
	if err != nil {
		RespondInternalError(c)
		return
	}

	RespondSuccess(c, http.StatusCreated, toGuestCodeDTO(guestCode), "Guest code berhasil dibuat")
}

// ListGuestCodes handles GET /api/admin/guest-codes
func (h *GuestHandler) ListGuestCodes(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * perPage

	codes, err := h.queries.ListGuestCodes(c.Request.Context(), repository.ListGuestCodesParams{
		Limit:  int32(perPage),
		Offset: int32(offset),
	})
	if err != nil {
		RespondInternalError(c)
		return
	}

	result := make([]gin.H, 0, len(codes))
	for _, code := range codes {
		loginCount, _ := h.queries.CountGuestCodeLogins(c.Request.Context(), code.ID)
		result = append(result, gin.H{
			"code":         toGuestCodeDTO(code),
			"total_logins": loginCount,
		})
	}

	RespondSuccess(c, http.StatusOK, result)
}

// GetGuestCodeDetail handles GET /api/admin/guest-codes/:id
func (h *GuestHandler) GetGuestCodeDetail(c *gin.Context) {
	codeID := stringToUUID(c.Param("id"))
	if !codeID.Valid {
		RespondBadRequest(c, "ID tidak valid")
		return
	}

	code, err := h.queries.GetGuestCodeByID(c.Request.Context(), codeID)
	if err != nil {
		RespondNotFound(c, "Guest code tidak ditemukan")
		return
	}

	// Get logins for this code
	logins, _ := h.queries.ListGuestLoginsByCode(c.Request.Context(), code.ID)
	loginDTOs := make([]guestLoginDTO, 0, len(logins))
	for _, l := range logins {
		loginDTOs = append(loginDTOs, toGuestLoginDTO(l))
	}

	RespondSuccess(c, http.StatusOK, gin.H{
		"code":   toGuestCodeDTO(code),
		"logins": loginDTOs,
	})
}

// RevokeGuestCode handles DELETE /api/admin/guest-codes/:id
func (h *GuestHandler) RevokeGuestCode(c *gin.Context) {
	codeID := stringToUUID(c.Param("id"))
	if !codeID.Valid {
		RespondBadRequest(c, "ID tidak valid")
		return
	}

	err := h.queries.DeactivateGuestCode(c.Request.Context(), codeID)
	if err != nil {
		RespondInternalError(c)
		return
	}

	RespondSuccess(c, http.StatusOK, nil, "Guest code berhasil direvoke")
}

// ── Helper ──────────────────────────────────────────────────────────

// generateGuestCode creates a code like "ATOM-A3X9"
func generateGuestCode() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no I,O,0,1 to avoid confusion
	code := make([]byte, 4)
	for i := range code {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		code[i] = chars[n.Int64()]
	}
	return fmt.Sprintf("ATOM-%s", string(code))
}
