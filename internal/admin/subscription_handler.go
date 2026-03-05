package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nununugraha/sains-api/internal/repository"
)

// ── Subscriptions ────────────────────────────────────────────────
// Template: templates/subscriptions.html
// SQL: subscriptions.sql
// Depends on: formatIDR, uuidStr (helpers.go)

type subRow struct {
	ID        string
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
				ID:        uuidStr(s.ID),
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
				ID:        uuidStr(s.ID),
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

// DeleteSubscription handles DELETE /admin/subscriptions/:id
func (h *AdminHandler) DeleteSubscription(c *gin.Context) {
	subID := parseUUID(c.Param("id"))
	_ = h.queries.DeleteSubscription(c.Request.Context(), subID)
	h.audit.Log(c, "delete_subscription", "subscription", c.Param("id"), "Subscription deleted")
	c.Status(http.StatusOK)
}
