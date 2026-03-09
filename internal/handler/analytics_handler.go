package handler

import (
	"crypto/sha256"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nununugraha/sains-api/internal/repository"
)

type AnalyticsHandler struct {
	queries *repository.Queries
}

func NewAnalyticsHandler(queries *repository.Queries) *AnalyticsHandler {
	return &AnalyticsHandler{queries: queries}
}

type PageViewRequest struct {
	URL         string `json:"url" binding:"required"`
	Referrer    string `json:"referrer"`
	UTMSource   string `json:"utm_source"`
	UTMMedium   string `json:"utm_medium"`
	UTMCampaign string `json:"utm_campaign"`
}

func (h *AnalyticsHandler) Collect(c *gin.Context) {
	var req PageViewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Simple IP hashing to count unique visitors without storing PII
	ip := c.ClientIP()
	hash := sha256.Sum256([]byte(ip + "sains-salt-2024"))
	ipHash := fmt.Sprintf("%x", hash)

	userAgent := c.Request.UserAgent()

	err := h.queries.RecordPageView(c.Request.Context(), repository.RecordPageViewParams{
		PageUrl:     req.URL,
		Referrer:    pgtype.Text{String: req.Referrer, Valid: req.Referrer != ""},
		IpHash:      ipHash,
		UserAgent:   pgtype.Text{String: userAgent, Valid: userAgent != ""},
		UtmSource:   pgtype.Text{String: req.UTMSource, Valid: req.UTMSource != ""},
		UtmMedium:   pgtype.Text{String: req.UTMMedium, Valid: req.UTMMedium != ""},
		UtmCampaign: pgtype.Text{String: req.UTMCampaign, Valid: req.UTMCampaign != ""},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record page view"})
		return
	}

	c.Status(http.StatusNoContent)
}
