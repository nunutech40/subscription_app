package admin

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nununugraha/sains-api/internal/repository"
)

// ── Settings (System Config) ─────────────────────────────────────
// Template: templates/settings.html
// SQL: system_config.sql
// Depends on: (none from helpers.go)

func (h *AdminHandler) Settings(c *gin.Context) {
	ctx := c.Request.Context()
	configs, _ := h.queries.ListConfigs(ctx)

	type configRow struct {
		Key         string
		Value       string
		Description string
		UpdatedAt   string
	}
	var rows []configRow
	for _, cfg := range configs {
		ua := ""
		if cfg.UpdatedAt.Valid {
			ua = cfg.UpdatedAt.Time.Format("02 Jan 15:04")
		}
		rows = append(rows, configRow{
			Key:         cfg.Key,
			Value:       cfg.Value,
			Description: cfg.Description.String,
			UpdatedAt:   ua,
		})
	}

	h.render(c, "settings", gin.H{
		"Title":   "System Settings",
		"active":  "settings",
		"Configs": rows,
	})
}

// UpdateConfig handles PUT /admin/settings/:key — inline edit config value
func (h *AdminHandler) UpdateConfig(c *gin.Context) {
	ctx := c.Request.Context()
	key := c.Param("key")
	value := c.PostForm("value")
	if value == "" {
		c.String(http.StatusBadRequest, "Value is required")
		return
	}

	err := h.queries.UpdateConfig(ctx, repository.UpdateConfigParams{
		Key:       key,
		Value:     value,
		UpdatedBy: pgtype.UUID{}, // no admin user tracking for now
	})
	if err != nil {
		log.Printf("update config error: %v", err)
		c.String(http.StatusInternalServerError, "Failed to update")
		return
	}

	h.audit.Log(c, "update_config", "system_config", key, fmt.Sprintf("Value changed to: %s", value))

	c.Header("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(c.Writer, `<span class="text-success"><i class="ti ti-check me-1"></i>%s</span>`, template.HTMLEscapeString(value))
}

// ── Products ─────────────────────────────────────────────────────
// Template: templates/products.html
// SQL: products.sql
// Depends on: (none from helpers.go)

func (h *AdminHandler) Products(c *gin.Context) {
	ctx := c.Request.Context()
	products, _ := h.queries.ListAllProducts(ctx)

	type productRow struct {
		ID          string
		Name        string
		Description string
		IsActive    bool
		CreatedAt   string
	}
	var rows []productRow
	for _, p := range products {
		ca := ""
		if p.CreatedAt.Valid {
			ca = p.CreatedAt.Time.Format("02 Jan 2006")
		}
		rows = append(rows, productRow{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description.String,
			IsActive:    p.IsActive.Bool,
			CreatedAt:   ca,
		})
	}

	h.render(c, "products", gin.H{
		"Title":    "Products",
		"active":   "products",
		"Products": rows,
	})
}

// CreateProduct handles POST /admin/products
func (h *AdminHandler) CreateProduct(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.PostForm("id")
	name := c.PostForm("name")
	desc := c.PostForm("description")

	if id == "" || name == "" {
		c.Redirect(http.StatusSeeOther, "/admin/products")
		return
	}

	_, err := h.queries.CreateProduct(ctx, repository.CreateProductParams{
		ID:          id,
		Name:        name,
		Description: pgtype.Text{String: desc, Valid: desc != ""},
		IsActive:    pgtype.Bool{Bool: true, Valid: true},
	})
	if err != nil {
		log.Printf("create product error: %v", err)
	} else {
		h.audit.Log(c, "create_product", "product", id, fmt.Sprintf("Created product: %s", name))
	}
	c.Redirect(http.StatusSeeOther, "/admin/products")
}

// UpdateProduct handles PUT /admin/products/:id
func (h *AdminHandler) UpdateProduct(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")
	name := c.PostForm("name")
	desc := c.PostForm("description")

	if name == "" {
		c.String(http.StatusBadRequest, "Name is required")
		return
	}

	err := h.queries.UpdateProduct(ctx, repository.UpdateProductParams{
		ID:          id,
		Name:        name,
		Description: pgtype.Text{String: desc, Valid: desc != ""},
	})
	if err != nil {
		log.Printf("update product error: %v", err)
		c.String(http.StatusInternalServerError, "Failed to update")
		return
	}
	c.Redirect(http.StatusSeeOther, "/admin/products")
}

// ToggleProductActive handles POST /admin/products/:id/toggle
func (h *AdminHandler) ToggleProductActive(c *gin.Context) {
	id := c.Param("id")
	err := h.queries.ToggleProductActive(c.Request.Context(), id)
	if err != nil {
		log.Printf("toggle product error: %v", err)
	}
	h.audit.Log(c, "toggle_product", "product", id, "Product active status toggled")
	c.Redirect(http.StatusSeeOther, "/admin/products")
}

// ── Audit Logs ───────────────────────────────────────────────────
// Template: templates/audit.html
// SQL: admin_audit.sql
// Depends on: (none from helpers.go)

func (h *AdminHandler) AuditLogs(c *gin.Context) {
	ctx := c.Request.Context()

	page := 1
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		page = p
	}
	perPage := int32(50)
	offset := int32((page - 1)) * perPage

	logs, _ := h.queries.ListAuditLogs(ctx, repository.ListAuditLogsParams{
		Limit:  perPage,
		Offset: offset,
	})
	total, _ := h.queries.CountAuditLogs(ctx)

	type auditRow struct {
		ID         int64
		AdminEmail string
		Action     string
		Resource   string
		ResourceID string
		Detail     string
		IP         string
		CreatedAt  string
	}
	var rows []auditRow
	for _, l := range logs {
		ip := ""
		if l.Ip != nil {
			ip = l.Ip.String()
		}
		ca := ""
		if l.CreatedAt.Valid {
			ca = l.CreatedAt.Time.Format("02 Jan 15:04:05")
		}
		rows = append(rows, auditRow{
			ID:         l.ID,
			AdminEmail: l.AdminEmail,
			Action:     l.Action,
			Resource:   l.Resource,
			ResourceID: l.ResourceID.String,
			Detail:     l.Detail.String,
			IP:         ip,
			CreatedAt:  ca,
		})
	}

	totalPages := int(total) / int(perPage)
	if int(total)%int(perPage) > 0 {
		totalPages++
	}

	h.render(c, "audit", gin.H{
		"Title":      "Audit Logs",
		"active":     "audit",
		"Logs":       rows,
		"Page":       page,
		"TotalPages": totalPages,
		"Total":      total,
		"HasPrev":    page > 1,
		"HasNext":    page < totalPages,
		"PrevPage":   page - 1,
		"NextPage":   page + 1,
	})
}
