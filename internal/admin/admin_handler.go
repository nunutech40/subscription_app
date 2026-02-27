package admin

import (
	"embed"
	"html/template"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/nununugraha/sains-api/internal/middleware"
	"github.com/nununugraha/sains-api/internal/repository"
	"github.com/nununugraha/sains-api/internal/service"
)

//go:embed templates/*.html
var templateFS embed.FS

// AdminHandler serves the HTMX-powered admin dashboard.
//
// This is the core struct — all handler methods are split into separate files:
//   - dashboard_handler.go       → Dashboard()
//   - audience_handler.go        → Audience()
//   - user_handler.go            → Users(), UserDetail(), LockUser(), UnlockUser()
//   - guest_code_handler.go      → GuestCodes(), GuestCodeDetail(), CreateGuestCode(), etc.
//   - subscription_handler.go    → Subscriptions()
//   - pricing_handler.go         → Pricing(), UpdatePriceInline(), TogglePlanActive()
//   - revenue_handler.go         → Revenue()
//   - anomaly_handler.go         → Anomalies()
//   - feedback_handler.go        → Feedback(), MarkFeedbackRead(), MarkAllFeedbackRead()
//   - settings_product_audit_handler.go → Settings(), Products(), AuditLogs()
//   - admin_auth_handler.go      → AdminAuthMiddleware(), LoginPage(), LoginPost(), Logout()
//   - helpers.go                 → formatIDR(), uuidStr(), parseUUID(), containsCI(), etc.
//
// See docs/ARCHITECTURE.md for the full modularity map.
type AdminHandler struct {
	queries      *repository.Queries
	authService  *service.AuthService
	tokenService *service.TokenService
	audit        *middleware.AuditLogger
	fs           embed.FS
}

// NewAdminHandler creates a new admin handler.
func NewAdminHandler(queries *repository.Queries, authService *service.AuthService, tokenService *service.TokenService, audit *middleware.AuditLogger) *AdminHandler {
	return &AdminHandler{queries: queries, authService: authService, tokenService: tokenService, audit: audit, fs: templateFS}
}

// ── Render helper ────────────────────────────────────────────────────

func (h *AdminHandler) render(c *gin.Context, tmplName string, data gin.H) {
	data["Active"] = data["active"]
	// Inject admin email from auth context
	if email, ok := c.Get("admin_email"); ok {
		data["AdminEmail"] = email
	}
	// Parse layout + specific page template to avoid "content" name collision
	tmpl := template.Must(template.ParseFS(h.fs, "templates/layout.html", "templates/"+tmplName+".html"))
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(c.Writer, "layout", data); err != nil {
		log.Printf("template error: %v", err)
		c.String(500, "Template error: "+err.Error())
	}
}
