package admin

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nununugraha/sains-api/internal/repository"
)

// ── Pricing ──────────────────────────────────────────────────────
// Template: templates/pricing.html
// SQL: pricing_plans.sql
// Depends on: formatIDR, uuidStr, parseUUID (helpers.go)

type planRow struct {
	ID           string
	ProductID    string
	Segment      string
	Duration     string
	DurationDays int32
	PriceIDR     string
	PriceRaw     int32
	Label        string
	IsActive     bool
}

type segmentGroup struct {
	Name  string
	Count int
	Plans []planRow
}

func (h *AdminHandler) Pricing(c *gin.Context) {
	ctx := c.Request.Context()

	// List ALL plans (including inactive) for admin management
	plans, _ := h.queries.ListAllPricingPlans(ctx)

	var planRows []planRow
	segmentMap := map[string][]planRow{}

	for _, p := range plans {
		pr := planRow{
			ID:           uuidStr(p.ID),
			ProductID:    p.ProductID.String,
			Segment:      p.Segment,
			Duration:     p.Duration,
			DurationDays: p.DurationDays,
			PriceIDR:     formatIDR(int64(p.PriceIdr)),
			PriceRaw:     p.PriceIdr,
			Label:        p.Label.String,
			IsActive:     p.IsActive.Bool,
		}
		planRows = append(planRows, pr)
		segmentMap[p.Segment] = append(segmentMap[p.Segment], pr)
	}

	var segments []segmentGroup
	for name, plans := range segmentMap {
		segments = append(segments, segmentGroup{
			Name:  name,
			Count: len(plans),
			Plans: plans,
		})
	}

	// Fetch products for the "Add Plan" form dropdown
	products, _ := h.queries.ListAllProducts(ctx)
	type productOption struct {
		ID   string
		Name string
	}
	var productOptions []productOption
	for _, p := range products {
		productOptions = append(productOptions, productOption{ID: p.ID, Name: p.Name})
	}

	// Flash message from redirect
	flash := c.Query("flash")

	h.render(c, "pricing", gin.H{
		"Title":    "Pricing Plans",
		"active":   "pricing",
		"Plans":    planRows,
		"Segments": segments,
		"Products": productOptions,
		"Flash":    flash,
	})
}

// CreatePlan handles POST /admin/pricing — tambah plan baru
func (h *AdminHandler) CreatePlan(c *gin.Context) {
	ctx := c.Request.Context()

	productID := c.PostForm("product_id")
	segment := c.PostForm("segment")
	duration := c.PostForm("duration")
	durationDaysStr := c.PostForm("duration_days")
	priceStr := c.PostForm("price_idr")
	label := c.PostForm("label")

	if productID == "" || segment == "" || duration == "" || durationDaysStr == "" || priceStr == "" {
		c.Redirect(http.StatusSeeOther, "/admin/pricing?flash=error_missing_fields")
		return
	}

	durationDays, err := strconv.Atoi(durationDaysStr)
	if err != nil || durationDays <= 0 {
		c.Redirect(http.StatusSeeOther, "/admin/pricing?flash=error_invalid_days")
		return
	}

	price, err := strconv.Atoi(priceStr)
	if err != nil || price < 0 {
		c.Redirect(http.StatusSeeOther, "/admin/pricing?flash=error_invalid_price")
		return
	}

	_, err = h.queries.CreatePricingPlan(ctx, repository.CreatePricingPlanParams{
		ProductID:    pgtype.Text{String: productID, Valid: true},
		Segment:      segment,
		Duration:     duration,
		DurationDays: int32(durationDays),
		PriceIdr:     int32(price),
		Label:        pgtype.Text{String: label, Valid: label != ""},
		IsActive:     pgtype.Bool{Bool: true, Valid: true},
	})
	if err != nil {
		log.Printf("create plan error: %v", err)
		c.Redirect(http.StatusSeeOther, "/admin/pricing?flash=error_create_failed")
		return
	}

	h.audit.Log(c, "create_plan", "pricing_plan", productID,
		fmt.Sprintf("Plan created: %s/%s/%s @ Rp %s", productID, segment, duration, formatIDR(int64(price))))
	c.Redirect(http.StatusSeeOther, "/admin/pricing?flash=success")
}

// UpdatePriceInline handles PUT /admin/pricing/:id — inline HTMX price edit
func (h *AdminHandler) UpdatePriceInline(c *gin.Context) {
	ctx := c.Request.Context()
	id := parseUUID(c.Param("id"))

	priceStr := c.PostForm("price")
	if priceStr == "" {
		c.String(http.StatusBadRequest, "Price is required")
		return
	}

	price, err := strconv.Atoi(priceStr)
	if err != nil || price < 0 {
		c.String(http.StatusBadRequest, "Invalid price")
		return
	}

	err = h.queries.UpdatePricingPlanPrice(ctx, repository.UpdatePricingPlanPriceParams{
		ID:       id,
		PriceIdr: int32(price),
	})
	if err != nil {
		log.Printf("update price error: %v", err)
		c.String(http.StatusInternalServerError, "Failed to update price")
		return
	}

	h.audit.Log(c, "update_price", "pricing_plan", c.Param("id"), fmt.Sprintf("Price updated to Rp %s", formatIDR(int64(price))))

	// Return updated formatted price for HTMX swap
	c.Header("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(c.Writer, `<span class="text-success"><i class="ti ti-check me-1"></i>Rp %s</span>`, formatIDR(int64(price)))
}

// TogglePlanActive handles POST /admin/pricing/:id/toggle
func (h *AdminHandler) TogglePlanActive(c *gin.Context) {
	id := parseUUID(c.Param("id"))
	err := h.queries.TogglePricingPlanActive(c.Request.Context(), id)
	if err != nil {
		log.Printf("toggle plan active error: %v", err)
	}
	h.audit.Log(c, "toggle_plan", "pricing_plan", c.Param("id"), "Plan active status toggled")
	c.Redirect(http.StatusSeeOther, "/admin/pricing")
}
