package admin

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nununugraha/sains-api/internal/repository"
)

// ── Feedback ─────────────────────────────────────────────────────
// Template: templates/feedback.html
// SQL: feedback.sql
// Depends on: formatIDR, uuidStr, parseUUID (helpers.go)

type feedbackRow struct {
	ID        int64
	Email     string
	Role      string
	Category  string
	Rating    int
	Message   string
	PageURL   string
	IsRead    bool
	CreatedAt string
}

// Feedback renders the admin feedback inbox page.
func (h *AdminHandler) Feedback(c *gin.Context) {
	ctx := c.Request.Context()

	// Get filter params
	category := c.Query("category")
	filter := c.DefaultQuery("filter", "all") // all, unread
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit := int32(50)
	offset := int32((page - 1)) * limit

	// Get stats
	stats, _ := h.queries.GetFeedbackStats(ctx)

	// Fetch feedback based on filter
	var feedbackList []repository.Feedback
	var err error

	if filter == "unread" {
		feedbackList, err = h.queries.ListUnreadFeedback(ctx, repository.ListUnreadFeedbackParams{
			Limit:  limit,
			Offset: offset,
		})
	} else if category != "" {
		feedbackList, err = h.queries.ListFeedbackByCategory(ctx, repository.ListFeedbackByCategoryParams{
			Category: category,
			Limit:    limit,
			Offset:   offset,
		})
	} else {
		feedbackList, err = h.queries.ListFeedback(ctx, repository.ListFeedbackParams{
			Limit:  limit,
			Offset: offset,
		})
	}

	if err != nil {
		log.Printf("list feedback error: %v", err)
	}

	var rows []feedbackRow
	for _, f := range feedbackList {
		rating := 0
		if f.Rating.Valid {
			rating = int(f.Rating.Int16)
		}
		pageURL := ""
		if f.PageUrl.Valid {
			pageURL = f.PageUrl.String
		}
		rows = append(rows, feedbackRow{
			ID:        f.ID,
			Email:     f.UserEmail,
			Role:      f.UserRole,
			Category:  f.Category,
			Rating:    rating,
			Message:   f.Message,
			PageURL:   pageURL,
			IsRead:    f.IsRead.Bool,
			CreatedAt: f.CreatedAt.Time.Format("2 Jan 2006 15:04"),
		})
	}

	// Format avg rating
	avgRating := "—"
	if stats.AvgRating.Valid {
		avgFloat, _ := stats.AvgRating.Float64Value()
		if avgFloat.Valid {
			avgRating = strconv.FormatFloat(avgFloat.Float64, 'f', 1, 64)
		}
	}

	h.render(c, "feedback", gin.H{
		"Title":       "Feedback",
		"active":      "feedback",
		"Feedbacks":   rows,
		"Stats":       stats,
		"AvgRating":   avgRating,
		"Filter":      filter,
		"Category":    category,
		"CurrentPage": page,
	})
}

// MarkFeedbackRead handles POST /admin/feedback/:id/read
func (h *AdminHandler) MarkFeedbackRead(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid ID")
		return
	}

	if err := h.queries.MarkFeedbackRead(c.Request.Context(), id); err != nil {
		log.Printf("mark feedback read error: %v", err)
		c.String(http.StatusInternalServerError, "Failed")
		return
	}

	h.audit.Log(c, "mark_read", "feedback", strconv.FormatInt(id, 10), "Feedback marked as read")

	// HTMX: return updated badge
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, `<span class="badge bg-success">Read</span>`)
}

// MarkAllFeedbackRead handles POST /admin/feedback/mark-all-read
func (h *AdminHandler) MarkAllFeedbackRead(c *gin.Context) {
	if err := h.queries.MarkAllFeedbackRead(c.Request.Context()); err != nil {
		log.Printf("mark all feedback read error: %v", err)
	}
	h.audit.Log(c, "mark_all_read", "feedback", "", "All feedback marked as read")
	c.Redirect(http.StatusSeeOther, "/admin/feedback")
}
