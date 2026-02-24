package admin

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nununugraha/sains-api/internal/middleware"
	"github.com/nununugraha/sains-api/internal/repository"
	"github.com/nununugraha/sains-api/internal/service"
)

//go:embed templates/*.html
var templateFS embed.FS

// AdminHandler serves the HTMX-powered admin dashboard.
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

// ── Helper types for templates ──────────────────────────────────────

type dashboardStats struct {
	TotalRevenue       string
	TotalSubscriptions int64
	ActiveSubscribers  int64
	MaxSubscribers     int
	SubsPercent        int
	ActiveGuests       int64
	MaxGuests          int
	GuestsPercent      int
	TotalUsers         int64
	FlaggedUsers       int
}

type subRow struct {
	Email     string
	ProductID string
	Segment   string
	Amount    string
	Status    string
	CreatedAt string
	UserID    string
	StartsAt  string
	ExpiresAt string
}

type anomalyRow struct {
	UserID        string
	Email         string
	Name          string
	Event         string
	LastEvent     string
	LastEventTime string
	ScoreDelta    int32
	TotalScore    int32
	Score         int32
	IsActive      bool
	Detail        string
	CreatedAt     string
}

type userRow struct {
	ID           string
	Email        string
	Name         string
	Role         string
	IsActive     bool
	AnomalyScore int32
	CreatedAt    string
}

type sessionRow struct {
	IP        string
	UserAgent string
	IsActive  bool
	ExpiresAt string
	CreatedAt string
}

type guestCodeRow struct {
	ID                string
	Code              string
	ProductID         string
	Label             string
	MaxLoginsPerEmail int32
	TotalLogins       int64
	LoginCount        int64
	IsActive          bool
	ExpiresAt         string
	CreatedAt         string
}

type guestLoginRow struct {
	Email          string
	LoginCount     int32
	LastLoginAt    string
	ReferralSource string
}

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

// ── Dashboard ─────────────────────────────────────────────────────

func (h *AdminHandler) Dashboard(c *gin.Context) {
	ctx := c.Request.Context()

	// Stats
	totalRevenue, _ := h.queries.GetTotalRevenue(ctx)
	totalSubs, _ := h.queries.CountAllSubscriptions(ctx)
	activeSubs, _ := h.queries.CountActiveSubscriptions(ctx)
	activeGuests, _ := h.queries.CountActiveGuestSessions(ctx)
	totalUsers, _ := h.queries.CountUsers(ctx)

	maxSubs := 200
	if cfg, err := h.queries.GetConfig(ctx, "max_subscribers"); err == nil {
		fmt.Sscanf(cfg.Value, "%d", &maxSubs)
	}
	maxGuests := 50
	if cfg, err := h.queries.GetConfig(ctx, "max_active_guests"); err == nil {
		fmt.Sscanf(cfg.Value, "%d", &maxGuests)
	}

	subsPercent := 0
	if maxSubs > 0 {
		subsPercent = int(activeSubs) * 100 / maxSubs
	}
	guestsPercent := 0
	if maxGuests > 0 {
		guestsPercent = int(activeGuests) * 100 / maxGuests
	}

	flagged, _ := h.queries.ListFlaggedUsers(ctx, repository.ListFlaggedUsersParams{
		ScoreDelta: 10, Limit: 100, Offset: 0,
	})

	stats := dashboardStats{
		TotalRevenue:       formatIDR(totalRevenue),
		TotalSubscriptions: totalSubs,
		ActiveSubscribers:  activeSubs,
		MaxSubscribers:     maxSubs,
		SubsPercent:        subsPercent,
		ActiveGuests:       activeGuests,
		MaxGuests:          maxGuests,
		GuestsPercent:      guestsPercent,
		TotalUsers:         totalUsers,
		FlaggedUsers:       len(flagged),
	}

	// Recent subscriptions
	recentSubs, _ := h.queries.ListAllSubscriptions(ctx, repository.ListAllSubscriptionsParams{
		Limit: 10, Offset: 0,
	})

	var subRows []subRow
	for _, s := range recentSubs {
		subRows = append(subRows, subRow{
			Email:     s.Email,
			ProductID: s.ProductID.String,
			Segment:   s.Segment, // string type
			Amount:    formatIDR(int64(s.AmountPaidIdr.Int32)),
			Status:    s.Status.String,
			CreatedAt: s.CreatedAt.Time.Format("02 Jan 15:04"),
		})
	}

	// Recent anomalies
	recentAnomalies, _ := h.queries.ListRecentAnomalies(ctx, repository.ListRecentAnomaliesParams{
		Limit: 5, Offset: 0,
	})

	var anomalyRows []anomalyRow
	for _, a := range recentAnomalies {
		anomalyRows = append(anomalyRows, anomalyRow{
			Email:      a.Email,
			Event:      a.Event,
			ScoreDelta: a.ScoreDelta,
			TotalScore: a.AnomalyScore.Int32,
			CreatedAt:  a.CreatedAt.Time.Format("02 Jan 15:04"),
		})
	}

	// Guest codes
	guestCodes, _ := h.queries.ListGuestCodes(ctx, repository.ListGuestCodesParams{
		Limit: 5, Offset: 0,
	})

	var codeRows []guestCodeRow
	for _, g := range guestCodes {
		logins, _ := h.queries.CountGuestCodeLogins(ctx, g.ID)
		codeRows = append(codeRows, guestCodeRow{
			Code:       g.Code,
			Label:      g.Label.String,
			LoginCount: logins,
			ExpiresAt:  g.ExpiresAt.Time.Format("02 Jan 15:04"),
		})
	}
	// Build segment chart data from subscription data
	segmentCounts := make(map[string]int)
	for _, s := range recentSubs {
		segmentCounts[s.Segment]++
	}
	chartLabels := []string{}
	chartValues := []int{}
	for seg, count := range segmentCounts {
		chartLabels = append(chartLabels, seg)
		chartValues = append(chartValues, count)
	}
	chartData := map[string]interface{}{
		"labels": chartLabels,
		"values": chartValues,
	}
	chartJSON, _ := json.Marshal(chartData)
	// Pass as string — it goes into data-chart="..." attribute (auto-escaped by html/template)
	segmentChartJSON := string(chartJSON)

	h.render(c, "dashboard", gin.H{
		"Title":            "Dashboard",
		"active":           "dashboard",
		"Stats":            stats,
		"RecentSubs":       subRows,
		"RecentAnomalies":  anomalyRows,
		"GuestCodes":       codeRows,
		"SegmentChartJSON": segmentChartJSON,
	})
}

// ── Users ────────────────────────────────────────────────────────

func (h *AdminHandler) Users(c *gin.Context) {
	ctx := c.Request.Context()
	query := c.DefaultQuery("q", "")
	roleFilter := c.DefaultQuery("role", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	perPage := int32(20)
	offset := int32(page-1) * perPage

	queryText := pgtype.Text{String: query, Valid: true}

	var users []repository.User
	var total int64

	if roleFilter != "" {
		users, _ = h.queries.SearchUsersByRole(ctx, repository.SearchUsersByRoleParams{
			Column1: queryText, Role: pgtype.Text{String: roleFilter, Valid: true},
			Limit: perPage, Offset: offset,
		})
		total, _ = h.queries.CountSearchUsersByRole(ctx, repository.CountSearchUsersByRoleParams{
			Column1: queryText, Role: pgtype.Text{String: roleFilter, Valid: true},
		})
	} else {
		users, _ = h.queries.SearchUsers(ctx, repository.SearchUsersParams{
			Column1: queryText, Limit: perPage, Offset: offset,
		})
		total, _ = h.queries.CountSearchUsers(ctx, queryText)
	}

	var userRows []userRow
	for _, u := range users {
		score, _ := h.queries.GetUserAnomalyScore(ctx, u.ID)
		userRows = append(userRows, userRow{
			ID:           uuidStr(u.ID),
			Email:        u.Email,
			Name:         u.Name.String,
			Role:         u.Role.String,
			IsActive:     u.IsActive.Bool,
			AnomalyScore: int32(score),
			CreatedAt:    u.CreatedAt.Time.Format("02 Jan 2006"),
		})
	}

	totalPages := int(total) / int(perPage)
	if int(total)%int(perPage) != 0 {
		totalPages++
	}

	h.render(c, "users", gin.H{
		"Title":      "Users",
		"active":     "users",
		"Users":      userRows,
		"Query":      query,
		"RoleFilter": roleFilter,
		"Page":       page,
		"TotalPages": totalPages,
		"Total":      total,
		"PrevPage":   page - 1,
		"NextPage":   page + 1,
	})
}

// ── User Detail ──────────────────────────────────────────────────

func (h *AdminHandler) UserDetail(c *gin.Context) {
	ctx := c.Request.Context()
	uid := parseUUID(c.Param("id"))

	user, err := h.queries.GetUserByID(ctx, uid)
	if err != nil {
		c.String(404, "User not found")
		return
	}

	score, _ := h.queries.GetUserAnomalyScore(ctx, uid)

	u := userRow{
		ID:           uuidStr(user.ID),
		Email:        user.Email,
		Name:         user.Name.String,
		Role:         user.Role.String,
		IsActive:     user.IsActive.Bool,
		AnomalyScore: int32(score),
		CreatedAt:    user.CreatedAt.Time.Format("02 Jan 2006 15:04"),
	}

	// Sessions
	sessions, _ := h.queries.ListSessionsByUser(ctx, uid)
	var sessRows []sessionRow
	for _, s := range sessions {
		ip := ""
		if s.IpAtLogin != nil {
			ip = s.IpAtLogin.String()
		}
		sessRows = append(sessRows, sessionRow{
			IP:        ip,
			UserAgent: s.UserAgent.String,
			IsActive:  s.IsActive.Bool,
			ExpiresAt: s.ExpiresAt.Time.Format("02 Jan 15:04"),
			CreatedAt: s.CreatedAt.Time.Format("02 Jan 15:04"),
		})
	}

	// Anomaly logs
	anomalyLogs, _ := h.queries.ListAnomalyLogsByUser(ctx, repository.ListAnomalyLogsByUserParams{
		UserID: uid, Limit: 20,
	})
	var aRows []anomalyRow
	for _, a := range anomalyLogs {
		detail := ""
		if len(a.Detail) > 0 {
			var d map[string]interface{}
			if json.Unmarshal(a.Detail, &d) == nil {
				if msg, ok := d["msg"]; ok {
					detail = fmt.Sprintf("%v", msg)
				}
			}
		}
		aRows = append(aRows, anomalyRow{
			Event:      a.Event,
			ScoreDelta: a.ScoreDelta,
			Detail:     detail,
			CreatedAt:  a.CreatedAt.Time.Format("02 Jan 15:04"),
		})
	}

	// Subscriptions
	subs, _ := h.queries.ListUserSubscriptions(ctx, uid)
	var sRows []subRow
	for _, s := range subs {
		sRows = append(sRows, subRow{
			ProductID: s.ProductID.String,
			Segment:   s.Segment,
			Amount:    formatIDR(int64(s.AmountPaidIdr.Int32)),
			Status:    s.Status.String,
			StartsAt:  s.StartsAt.Time.Format("02 Jan 2006"),
			ExpiresAt: s.ExpiresAt.Time.Format("02 Jan 2006"),
		})
	}

	h.render(c, "user_detail", gin.H{
		"Title":         "User: " + user.Email,
		"active":        "users",
		"User":          u,
		"Sessions":      sessRows,
		"AnomalyLogs":   aRows,
		"Subscriptions": sRows,
	})
}

// ── Lock / Unlock ────────────────────────────────────────────────

func (h *AdminHandler) LockUser(c *gin.Context) {
	uid := parseUUID(c.Param("id"))
	_ = h.queries.SetUserActive(c.Request.Context(), repository.SetUserActiveParams{
		IsActive: pgtype.Bool{Bool: false, Valid: true}, ID: uid,
	})
	_ = h.queries.RevokeAllUserSessions(c.Request.Context(), uid)
	h.audit.Log(c, "lock_user", "user", c.Param("id"), "User locked and all sessions revoked")
	c.Redirect(http.StatusSeeOther, "/admin/users/"+c.Param("id"))
}

func (h *AdminHandler) UnlockUser(c *gin.Context) {
	uid := parseUUID(c.Param("id"))
	_ = h.queries.SetUserActive(c.Request.Context(), repository.SetUserActiveParams{
		IsActive: pgtype.Bool{Bool: true, Valid: true}, ID: uid,
	})
	h.audit.Log(c, "unlock_user", "user", c.Param("id"), "User unlocked")
	c.Redirect(http.StatusSeeOther, "/admin/users/"+c.Param("id"))
}

// ── Anomalies ────────────────────────────────────────────────────

func (h *AdminHandler) Anomalies(c *gin.Context) {
	ctx := c.Request.Context()
	flagged, _ := h.queries.ListFlaggedUsers(ctx, repository.ListFlaggedUsersParams{
		ScoreDelta: 10, Limit: 50, Offset: 0,
	})

	var rows []anomalyRow
	for _, f := range flagged {
		// Get last event
		logs, _ := h.queries.ListAnomalyLogsByUser(ctx, repository.ListAnomalyLogsByUserParams{
			UserID: f.ID, Limit: 1,
		})
		lastEvent := ""
		lastTime := ""
		if len(logs) > 0 {
			lastEvent = logs[0].Event
			lastTime = logs[0].CreatedAt.Time.Format("02 Jan 15:04")
		}

		rows = append(rows, anomalyRow{
			UserID:        uuidStr(f.ID),
			Email:         f.Email,
			Name:          f.Name.String,
			Score:         f.TotalScore,
			LastEvent:     lastEvent,
			LastEventTime: lastTime,
			IsActive:      f.IsActive.Bool,
		})
	}

	h.render(c, "anomalies", gin.H{
		"Title":     "Anomalies",
		"active":    "anomalies",
		"Anomalies": rows,
	})
}

// ── Guest Codes ──────────────────────────────────────────────────

func (h *AdminHandler) GuestCodes(c *gin.Context) {
	ctx := c.Request.Context()
	codes, _ := h.queries.ListGuestCodes(ctx, repository.ListGuestCodesParams{
		Limit: 50, Offset: 0,
	})

	var rows []guestCodeRow
	for _, g := range codes {
		logins, _ := h.queries.CountGuestCodeLogins(ctx, g.ID)
		rows = append(rows, guestCodeRow{
			ID:                uuidStr(g.ID),
			Code:              g.Code,
			ProductID:         g.ProductID.String,
			Label:             g.Label.String,
			MaxLoginsPerEmail: g.MaxLoginsPerEmail.Int32,
			TotalLogins:       logins,
			IsActive:          g.IsActive.Bool,
			ExpiresAt:         g.ExpiresAt.Time.Format("02 Jan 2006 15:04"),
			CreatedAt:         g.CreatedAt.Time.Format("02 Jan 2006"),
		})
	}

	h.render(c, "guest_codes", gin.H{
		"Title":  "Guest Codes",
		"active": "guest-codes",
		"Codes":  rows,
	})
}

func (h *AdminHandler) GuestCodeDetail(c *gin.Context) {
	ctx := c.Request.Context()
	gid := parseUUID(c.Param("id"))

	code, err := h.queries.GetGuestCodeByID(ctx, gid)
	if err != nil {
		c.String(404, "Code not found")
		return
	}

	gc := guestCodeRow{
		ID:                uuidStr(code.ID),
		Code:              code.Code,
		ProductID:         code.ProductID.String,
		Label:             code.Label.String,
		MaxLoginsPerEmail: code.MaxLoginsPerEmail.Int32,
		IsActive:          code.IsActive.Bool,
		ExpiresAt:         code.ExpiresAt.Time.Format("02 Jan 2006 15:04"),
		CreatedAt:         code.CreatedAt.Time.Format("02 Jan 2006"),
	}

	logins, _ := h.queries.ListGuestLoginsByCode(ctx, gid)
	var loginRows []guestLoginRow
	for _, l := range logins {
		loginRows = append(loginRows, guestLoginRow{
			Email:          l.Email,
			LoginCount:     l.LoginCount.Int32,
			LastLoginAt:    l.LastLoginAt.Time.Format("02 Jan 2006 15:04"),
			ReferralSource: l.ReferralSource.String,
		})
	}

	h.render(c, "guest_code_detail", gin.H{
		"Title":  "Guest Code: " + code.Code,
		"active": "guest-codes",
		"Code":   gc,
		"Logins": loginRows,
	})
}

func (h *AdminHandler) CreateGuestCode(c *gin.Context) {
	label := c.PostForm("label")
	maxLogins, _ := strconv.Atoi(c.DefaultPostForm("max_logins", "2"))
	expiresHours, _ := strconv.Atoi(c.DefaultPostForm("expires_hours", "48"))

	code := generateGuestCode()

	_, err := h.queries.CreateGuestCode(c.Request.Context(), repository.CreateGuestCodeParams{
		Code:              code,
		ProductID:         pgtype.Text{String: "atomic", Valid: true},
		Label:             pgtype.Text{String: label, Valid: true},
		MaxLoginsPerEmail: pgtype.Int4{Int32: int32(maxLogins), Valid: true},
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(time.Duration(expiresHours) * time.Hour),
			Valid: true,
		},
	})
	if err != nil {
		log.Printf("create guest code error: %v", err)
	}

	c.Redirect(http.StatusSeeOther, "/admin/guest-codes")
}

func (h *AdminHandler) RevokeGuestCode(c *gin.Context) {
	gid := parseUUID(c.Param("id"))
	_ = h.queries.DeactivateGuestCode(c.Request.Context(), gid)

	// Re-fetch the code so we can return the updated row for HTMX swap
	gc, err := h.queries.GetGuestCodeByID(c.Request.Context(), gid)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/admin/guest-codes")
		return
	}
	cnt, _ := h.queries.CountGuestCodeLogins(c.Request.Context(), gid)

	row := guestCodeRow{
		ID:                uuidStr(gc.ID),
		Code:              gc.Code,
		ProductID:         gc.ProductID.String,
		Label:             gc.Label.String,
		MaxLoginsPerEmail: gc.MaxLoginsPerEmail.Int32,
		TotalLogins:       cnt,
		IsActive:          gc.IsActive.Bool,
		ExpiresAt:         gc.ExpiresAt.Time.Format("02 Jan 2006 15:04"),
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, `<tr id="code-`+row.ID+`">
		<td><code class="fs-5">`+row.Code+`</code></td>
		<td>`+row.Label+`</td>
		<td>`+row.ProductID+`</td>
		<td>`+fmt.Sprintf("%d", row.MaxLoginsPerEmail)+`</td>
		<td>`+fmt.Sprintf("%d", row.TotalLogins)+`</td>
		<td><span class="badge bg-red">Revoked</span></td>
		<td>`+row.ExpiresAt+`</td>
		<td>
			<a href="/admin/guest-codes/`+row.ID+`" class="btn btn-sm btn-outline-primary me-1">
				<i class="ti ti-eye"></i>
			</a>
			<button class="btn btn-sm btn-outline-danger"
				hx-delete="/admin/guest-codes/`+row.ID+`"
				hx-target="#code-`+row.ID+`" hx-swap="outerHTML"
				hx-confirm="Hapus kode `+row.Code+` secara permanen?">
				<i class="ti ti-trash"></i>
			</button>
		</td>
	</tr>`)
}

func (h *AdminHandler) DeleteGuestCode(c *gin.Context) {
	gid := parseUUID(c.Param("id"))
	_ = h.queries.DeleteGuestCode(c.Request.Context(), gid)
	// Return empty string — HTMX will remove the row
	c.String(http.StatusOK, "")
}

// ── Subscriptions ────────────────────────────────────────────────

func (h *AdminHandler) Subscriptions(c *gin.Context) {
	ctx := c.Request.Context()
	statusFilter := c.DefaultQuery("status", "")

	var rows []subRow

	if statusFilter != "" {
		filtered, _ := h.queries.ListSubscriptionsByStatus(ctx, repository.ListSubscriptionsByStatusParams{
			Status: pgtype.Text{String: statusFilter, Valid: true},
			Limit:  50, Offset: 0,
		})
		for _, s := range filtered {
			rows = append(rows, subRow{
				UserID:    uuidStr(s.UserID),
				Email:     s.Email,
				ProductID: s.ProductID.String,
				Segment:   s.Segment,
				Amount:    formatIDR(int64(s.AmountPaidIdr.Int32)),
				Status:    s.Status.String,
				StartsAt:  s.StartsAt.Time.Format("02 Jan 2006"),
				ExpiresAt: s.ExpiresAt.Time.Format("02 Jan 2006"),
			})
		}
	} else {
		subs, _ := h.queries.ListAllSubscriptions(ctx, repository.ListAllSubscriptionsParams{
			Limit: 50, Offset: 0,
		})
		for _, s := range subs {
			rows = append(rows, subRow{
				UserID:    uuidStr(s.UserID),
				Email:     s.Email,
				ProductID: s.ProductID.String,
				Segment:   s.Segment,
				Amount:    formatIDR(int64(s.AmountPaidIdr.Int32)),
				Status:    s.Status.String,
				StartsAt:  s.StartsAt.Time.Format("02 Jan 2006"),
				ExpiresAt: s.ExpiresAt.Time.Format("02 Jan 2006"),
			})
		}
	}

	h.render(c, "subscriptions", gin.H{
		"Title":         "Subscriptions",
		"active":        "subscriptions",
		"Subscriptions": rows,
		"StatusFilter":  statusFilter,
	})
}

// ── Pricing ──────────────────────────────────────────────────────

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

	h.render(c, "pricing", gin.H{
		"Title":    "Pricing Plans",
		"active":   "pricing",
		"Plans":    planRows,
		"Segments": segments,
	})
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

// ── Revenue Analytics ────────────────────────────────────────────

func (h *AdminHandler) Revenue(c *gin.Context) {
	ctx := c.Request.Context()

	// Total revenue
	totalRevenue, _ := h.queries.GetTotalRevenue(ctx)
	activeSubs, _ := h.queries.CountActiveSubscriptions(ctx)
	totalSubs, _ := h.queries.CountAllSubscriptions(ctx)

	// Monthly revenue for line chart
	monthlyRevenue, _ := h.queries.GetMonthlyRevenue(ctx)
	revenueLabels := []string{}
	revenueValues := []int64{}
	for _, m := range monthlyRevenue {
		revenueLabels = append(revenueLabels, m.Month)
		revenueValues = append(revenueValues, m.Revenue)
	}
	revChartData := map[string]interface{}{
		"labels": revenueLabels,
		"values": revenueValues,
	}
	revChartJSON, _ := json.Marshal(revChartData)

	// Revenue by segment for doughnut chart
	segmentRevenue, _ := h.queries.GetRevenueBySegment(ctx)
	segLabels := []string{}
	segValues := []int64{}
	segCounts := []int64{}
	for _, s := range segmentRevenue {
		segLabels = append(segLabels, s.Segment)
		segValues = append(segValues, s.Revenue)
		segCounts = append(segCounts, s.Count)
	}
	segChartData := map[string]interface{}{
		"labels": segLabels,
		"values": segValues,
		"counts": segCounts,
	}
	segChartJSON, _ := json.Marshal(segChartData)

	// Monthly subscription counts for area chart
	monthlySubs, _ := h.queries.GetMonthlySubscriptionCount(ctx)
	subMonths := []string{}
	subActive := []int64{}
	subExpired := []int64{}
	subPending := []int64{}
	for _, m := range monthlySubs {
		subMonths = append(subMonths, m.Month)
		subActive = append(subActive, m.Active)
		subExpired = append(subExpired, m.Expired)
		subPending = append(subPending, m.Pending)
	}
	subChartData := map[string]interface{}{
		"months":  subMonths,
		"active":  subActive,
		"expired": subExpired,
		"pending": subPending,
	}
	subChartJSON, _ := json.Marshal(subChartData)

	// Build segment table data
	type segRow struct {
		Segment  string
		Revenue  string
		Count    int64
		AvgValue string
	}
	var segRows []segRow
	for _, s := range segmentRevenue {
		avg := int64(0)
		if s.Count > 0 {
			avg = s.Revenue / s.Count
		}
		segRows = append(segRows, segRow{
			Segment:  s.Segment,
			Revenue:  formatIDR(s.Revenue),
			Count:    s.Count,
			AvgValue: formatIDR(avg),
		})
	}

	h.render(c, "revenue", gin.H{
		"Title":            "Revenue Analytics",
		"active":           "revenue",
		"TotalRevenue":     formatIDR(totalRevenue),
		"ActiveSubs":       activeSubs,
		"TotalSubs":        totalSubs,
		"RevenueChartJSON": string(revChartJSON),
		"SegmentChartJSON": string(segChartJSON),
		"SubChartJSON":     string(subChartJSON),
		"SegmentRows":      segRows,
	})
}

// ── Settings (System Config) ─────────────────────────────────────

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

// ── Helpers ──────────────────────────────────────────────────────

func formatIDR(amount int64) string {
	s := fmt.Sprintf("%d", amount)
	if len(s) <= 3 {
		return s
	}
	result := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += "."
		}
		result += string(c)
	}
	return result
}

func uuidStr(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func parseUUID(s string) pgtype.UUID {
	var u pgtype.UUID
	u.Scan(s)
	return u
}

func generateGuestCode() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 4)
	for i := range b {
		b[i] = chars[time.Now().UnixNano()%int64(len(chars))]
		time.Sleep(1 * time.Nanosecond)
	}
	return "ATOM-" + string(b)
}

// ── Admin Auth ───────────────────────────────────────────────────

// AdminAuthMiddleware checks for admin_token cookie with valid admin JWT.
func (h *AdminHandler) AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr, err := c.Cookie("admin_token")
		if err != nil || tokenStr == "" {
			c.Redirect(http.StatusFound, "/admin/login")
			c.Abort()
			return
		}

		claims, err := h.tokenService.ValidateToken(tokenStr)
		if err != nil {
			// Clear invalid cookie
			c.SetCookie("admin_token", "", -1, "/admin", "", false, true)
			c.Redirect(http.StatusFound, "/admin/login")
			c.Abort()
			return
		}

		if claims.Role != "admin" {
			c.SetCookie("admin_token", "", -1, "/admin", "", false, true)
			c.Redirect(http.StatusFound, "/admin/login")
			c.Abort()
			return
		}

		// Set admin info in context
		c.Set("admin_id", claims.UserID)
		c.Set("admin_email", claims.Email)
		c.Next()
	}
}

// LoginPage serves GET /admin/login
func (h *AdminHandler) LoginPage(c *gin.Context) {
	// If already authenticated, redirect to dashboard
	if tokenStr, err := c.Cookie("admin_token"); err == nil && tokenStr != "" {
		if claims, err := h.tokenService.ValidateToken(tokenStr); err == nil && claims.Role == "admin" {
			c.Redirect(http.StatusFound, "/admin/")
			return
		}
	}

	tmpl := template.Must(template.ParseFS(h.fs, "templates/login.html"))
	c.Header("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(c.Writer, gin.H{"Error": "", "Email": ""})
}

// LoginPost handles POST /admin/login
func (h *AdminHandler) LoginPost(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")

	result, err := h.authService.Login(c.Request.Context(), service.LoginInput{
		Email:    email,
		Password: password,
	})
	if err != nil {
		tmpl := template.Must(template.ParseFS(h.fs, "templates/login.html"))
		c.Header("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(c.Writer, gin.H{"Error": "Email atau password salah.", "Email": email})
		return
	}

	// Check admin role
	if result.User.Role.String != "admin" {
		tmpl := template.Must(template.ParseFS(h.fs, "templates/login.html"))
		c.Header("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(c.Writer, gin.H{"Error": "Hanya akun admin yang bisa mengakses dashboard.", "Email": email})
		return
	}

	// Generate JWT for admin session
	token, err := h.tokenService.GenerateAccessToken(
		uuidStr(result.User.ID),
		result.User.Email,
		result.User.Role.String,
	)
	if err != nil {
		log.Printf("admin token error: %v", err)
		tmpl := template.Must(template.ParseFS(h.fs, "templates/login.html"))
		c.Header("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(c.Writer, gin.H{"Error": "Gagal membuat sesi. Coba lagi.", "Email": email})
		return
	}

	// Set httpOnly cookie (24 hours)
	c.SetCookie("admin_token", token, 24*60*60, "/admin", "", false, true)
	c.Redirect(http.StatusFound, "/admin/")
}

// Logout handles GET /admin/logout
func (h *AdminHandler) Logout(c *gin.Context) {
	c.SetCookie("admin_token", "", -1, "/admin", "", false, true)
	c.Redirect(http.StatusFound, "/admin/login")
}
