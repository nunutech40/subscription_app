package admin

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nununugraha/sains-api/internal/repository"
)

// ── Guest Codes ──────────────────────────────────────────────────
// Template: templates/guest_codes.html, templates/guest_code_detail.html
// SQL: guest_codes.sql
// Depends on: uuidStr, parseUUID, generateGuestCode (helpers.go)

type guestCodeRow struct {
	ID                string
	Code              string
	ProductID         string
	Label             string
	MaxLoginsPerEmail int32
	UniqueEmails      int64
	TotalLogins       int64
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

func (h *AdminHandler) GuestCodes(c *gin.Context) {
	ctx := c.Request.Context()
	codes, _ := h.queries.ListGuestCodes(ctx, repository.ListGuestCodesParams{
		Limit: 50, Offset: 0,
	})

	var rows []guestCodeRow
	for _, g := range codes {
		uniqueEmails, _ := h.queries.CountGuestCodeLogins(ctx, g.ID)
		totalLogins, _ := h.queries.SumGuestCodeLogins(ctx, g.ID)
		rows = append(rows, guestCodeRow{
			ID:                uuidStr(g.ID),
			Code:              g.Code,
			ProductID:         g.ProductID.String,
			Label:             g.Label.String,
			MaxLoginsPerEmail: g.MaxLoginsPerEmail.Int32,
			UniqueEmails:      uniqueEmails,
			TotalLogins:       totalLogins,
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
	totalLogins, _ := h.queries.SumGuestCodeLogins(c.Request.Context(), gid)

	row := guestCodeRow{
		ID:                uuidStr(gc.ID),
		Code:              gc.Code,
		ProductID:         gc.ProductID.String,
		Label:             gc.Label.String,
		MaxLoginsPerEmail: gc.MaxLoginsPerEmail.Int32,
		UniqueEmails:      cnt,
		TotalLogins:       totalLogins,
		IsActive:          gc.IsActive.Bool,
		ExpiresAt:         gc.ExpiresAt.Time.Format("02 Jan 2006 15:04"),
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, `<tr id="code-`+row.ID+`">
		<td><code class="fs-5">`+row.Code+`</code></td>
		<td>`+row.Label+`</td>
		<td>`+row.ProductID+`</td>
		<td>`+fmt.Sprintf("%d", row.MaxLoginsPerEmail)+`</td>
		<td>`+fmt.Sprintf("%d emails · %d logins", row.UniqueEmails, row.TotalLogins)+`</td>
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
