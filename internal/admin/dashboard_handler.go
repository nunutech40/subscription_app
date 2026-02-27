package admin

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/nununugraha/sains-api/internal/repository"
)

// ── Dashboard ─────────────────────────────────────────────────────
// Template: templates/dashboard.html
// SQL: multiple read-only stats queries
// Depends on: formatIDR (helpers.go)

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

func (h *AdminHandler) Dashboard(c *gin.Context) {
	ctx := c.Request.Context()

	// Stats — log errors so failures are visible
	totalRevenue, err := h.queries.GetTotalRevenue(ctx)
	if err != nil {
		log.Printf("[Dashboard] GetTotalRevenue error: %v", err)
	}
	totalSubs, err := h.queries.CountAllSubscriptions(ctx)
	if err != nil {
		log.Printf("[Dashboard] CountAllSubscriptions error: %v", err)
	}
	activeSubs, err := h.queries.CountActiveSubscriptions(ctx)
	if err != nil {
		log.Printf("[Dashboard] CountActiveSubscriptions error: %v", err)
	}
	activeGuests, err := h.queries.CountActiveGuestSessions(ctx)
	if err != nil {
		log.Printf("[Dashboard] CountActiveGuestSessions error: %v", err)
	}
	totalUsers, err := h.queries.CountUsers(ctx)
	if err != nil {
		log.Printf("[Dashboard] CountUsers error: %v", err)
	}

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
		uniqueEmails, _ := h.queries.CountGuestCodeLogins(ctx, g.ID)
		totalLogins, _ := h.queries.SumGuestCodeLogins(ctx, g.ID)
		codeRows = append(codeRows, guestCodeRow{
			Code:         g.Code,
			Label:        g.Label.String,
			UniqueEmails: uniqueEmails,
			TotalLogins:  totalLogins,
			ExpiresAt:    g.ExpiresAt.Time.Format("02 Jan 15:04"),
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
