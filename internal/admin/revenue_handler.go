package admin

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
)

// ── Revenue Analytics ────────────────────────────────────────────
// Template: templates/revenue.html
// SQL: subscriptions.sql (GetTotalRevenue, GetMonthlyRevenue, etc.)
// Depends on: formatIDR (helpers.go)

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
