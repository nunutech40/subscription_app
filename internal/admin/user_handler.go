package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nununugraha/sains-api/internal/repository"
)

// ── Users ────────────────────────────────────────────────────────
// Template: templates/users.html, templates/user_detail.html
// SQL: users.sql, sessions.sql, anomaly_logs.sql, subscriptions.sql
// Depends on: formatIDR, uuidStr, parseUUID, containsCI (helpers.go)

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

// ── Delete User ──────────────────────────────────────────────────

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	uid := parseUUID(c.Param("id"))

	// Prevent deleting admin users
	user, err := h.queries.GetUserByID(c.Request.Context(), uid)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/admin/users")
		return
	}
	if user.Role.String == "admin" {
		c.String(http.StatusForbidden, "Cannot delete admin users")
		return
	}

	_ = h.queries.DeleteUser(c.Request.Context(), uid)
	h.audit.Log(c, "delete_user", "user", c.Param("id"),
		fmt.Sprintf("User deleted: %s (%s)", user.Email, user.Name.String))
	c.Redirect(http.StatusSeeOther, "/admin/users")
}
