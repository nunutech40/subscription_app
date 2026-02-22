package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nununugraha/sains-api/internal/repository"
)

// PlanHandler handles pricing plan endpoints.
type PlanHandler struct {
	queries *repository.Queries
}

// NewPlanHandler creates a new PlanHandler.
func NewPlanHandler(queries *repository.Queries) *PlanHandler {
	return &PlanHandler{queries: queries}
}

// ── DTOs ────────────────────────────────────────────────────────────

type planDTO struct {
	ID           string `json:"id"`
	ProductID    string `json:"product_id"`
	Segment      string `json:"segment"`
	Duration     string `json:"duration"`
	DurationDays int    `json:"duration_days"`
	PriceIDR     int    `json:"price_idr"`
	Label        string `json:"label"`
}

func toPlanDTO(p repository.PricingPlan) planDTO {
	return planDTO{
		ID:           uuidToString(p.ID),
		ProductID:    p.ProductID.String,
		Segment:      p.Segment,
		Duration:     p.Duration,
		DurationDays: int(p.DurationDays),
		PriceIDR:     int(p.PriceIdr),
		Label:        p.Label.String,
	}
}

// ── Handlers ────────────────────────────────────────────────────────

// ListPlans handles GET /api/plans?product=atomic&segment=student
func (h *PlanHandler) ListPlans(c *gin.Context) {
	productID := c.DefaultQuery("product", "atomic")

	plans, err := h.queries.ListPricingPlansByProduct(c.Request.Context(),
		pgtype.Text{String: productID, Valid: true})
	if err != nil {
		RespondInternalError(c)
		return
	}

	// Filter by segment if specified
	segment := c.Query("segment")
	var result []planDTO
	for _, p := range plans {
		if segment == "" || p.Segment == segment {
			result = append(result, toPlanDTO(p))
		}
	}

	if result == nil {
		result = []planDTO{} // empty array, not null
	}

	RespondSuccess(c, http.StatusOK, result)
}

// GetPlan handles GET /api/plans/:id
func (h *PlanHandler) GetPlan(c *gin.Context) {
	planID := stringToUUID(c.Param("id"))
	if !planID.Valid {
		RespondBadRequest(c, "Plan ID tidak valid")
		return
	}

	plan, err := h.queries.GetPricingPlan(c.Request.Context(), planID)
	if err != nil {
		RespondNotFound(c, "Plan tidak ditemukan")
		return
	}

	RespondSuccess(c, http.StatusOK, toPlanDTO(plan))
}

// CreatePlan handles POST /admin/pricing-plans (admin only)
func (h *PlanHandler) CreatePlan(c *gin.Context) {
	var req struct {
		ProductID    string `json:"product_id" binding:"required"`
		Segment      string `json:"segment" binding:"required,oneof=global student parent"`
		Duration     string `json:"duration" binding:"required,oneof=monthly 3month 6month yearly"`
		DurationDays int    `json:"duration_days" binding:"required,min=1"`
		PriceIDR     int    `json:"price_idr" binding:"required,min=0"`
		Label        string `json:"label"`
		IsActive     *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondBadRequest(c, "Data tidak valid: "+err.Error())
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	plan, err := h.queries.CreatePricingPlan(c.Request.Context(), repository.CreatePricingPlanParams{
		ProductID:    pgtype.Text{String: req.ProductID, Valid: true},
		Segment:      req.Segment,
		Duration:     req.Duration,
		DurationDays: int32(req.DurationDays),
		PriceIdr:     int32(req.PriceIDR),
		Label:        pgtype.Text{String: req.Label, Valid: req.Label != ""},
		IsActive:     pgtype.Bool{Bool: isActive, Valid: true},
	})
	if err != nil {
		RespondInternalError(c)
		return
	}

	RespondSuccess(c, http.StatusCreated, toPlanDTO(plan), "Plan berhasil dibuat")
}

// UpdatePlan handles PUT /admin/pricing-plans/:id (admin only)
func (h *PlanHandler) UpdatePlan(c *gin.Context) {
	planID := stringToUUID(c.Param("id"))
	if !planID.Valid {
		RespondBadRequest(c, "Plan ID tidak valid")
		return
	}

	var req struct {
		PriceIDR int    `json:"price_idr" binding:"required,min=0"`
		Label    string `json:"label"`
		IsActive *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondBadRequest(c, "Data tidak valid: "+err.Error())
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	err := h.queries.UpdatePricingPlan(c.Request.Context(), repository.UpdatePricingPlanParams{
		ID:       planID,
		PriceIdr: int32(req.PriceIDR),
		Label:    pgtype.Text{String: req.Label, Valid: req.Label != ""},
		IsActive: pgtype.Bool{Bool: isActive, Valid: true},
	})
	if err != nil {
		RespondInternalError(c)
		return
	}

	RespondSuccess(c, http.StatusOK, nil, "Plan berhasil diupdate")
}
