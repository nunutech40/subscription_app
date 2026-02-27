package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/nununugraha/sains-api/internal/repository"
)

// ── Anomalies ────────────────────────────────────────────────────
// Template: templates/anomalies.html
// SQL: anomaly_logs.sql, users.sql
// Depends on: uuidStr (helpers.go)

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
