package handler

import (
	"net/http"
	"net/netip"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nununugraha/sains-api/internal/repository"
)

// FeedbackHandler handles feedback/suggestion-box endpoints.
type FeedbackHandler struct {
	queries *repository.Queries
}

// NewFeedbackHandler creates a new FeedbackHandler.
func NewFeedbackHandler(queries *repository.Queries) *FeedbackHandler {
	return &FeedbackHandler{queries: queries}
}

// ── Request types ────────────────────────────────────────────────────

type createFeedbackRequest struct {
	Category string `json:"category" binding:"required,oneof=saran bug pertanyaan"`
	Rating   *int16 `json:"rating"` // 1-5, optional
	Message  string `json:"message" binding:"required,min=3,max=2000"`
	PageURL  string `json:"page_url"` // which page user was on
}

// ── Submit Feedback (public — after login) ───────────────────────────

func (h *FeedbackHandler) Submit(c *gin.Context) {
	var req createFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondBadRequest(c, "Data tidak valid: "+err.Error())
		return
	}

	// Get user info from auth context
	email, _ := c.Get("email")
	role, _ := c.Get("role")

	emailStr := ""
	if email != nil {
		emailStr = email.(string)
	}
	roleStr := "guest"
	if role != nil {
		roleStr = role.(string)
	}

	// Parse rating
	var rating pgtype.Int2
	if req.Rating != nil && *req.Rating >= 1 && *req.Rating <= 5 {
		rating = pgtype.Int2{Int16: *req.Rating, Valid: true}
	}

	// Parse IP
	clientIP := c.ClientIP()
	var ip *netip.Addr
	if parsed, err := netip.ParseAddr(clientIP); err == nil {
		ip = &parsed
	}

	// Sanitize
	category := strings.ToLower(strings.TrimSpace(req.Category))
	if category != "saran" && category != "bug" && category != "pertanyaan" {
		category = "saran"
	}

	ctx := c.Request.Context()
	feedback, err := h.queries.CreateFeedback(ctx, repository.CreateFeedbackParams{
		UserEmail: emailStr,
		UserRole:  roleStr,
		Category:  category,
		Rating:    rating,
		Message:   strings.TrimSpace(req.Message),
		PageUrl:   pgtype.Text{String: req.PageURL, Valid: req.PageURL != ""},
		UserAgent: pgtype.Text{String: c.GetHeader("User-Agent"), Valid: true},
		Ip:        ip,
	})
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrCodeInternal, "Gagal menyimpan feedback")
		return
	}

	RespondSuccess(c, http.StatusCreated, gin.H{
		"id":         feedback.ID,
		"category":   feedback.Category,
		"created_at": feedback.CreatedAt,
	}, "Terima kasih atas feedback-nya! 🙏")
}
