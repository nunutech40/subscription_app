package admin

import (
	"fmt"
	"html/template"
	"log"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nununugraha/sains-api/internal/repository"
)

// ── Audience (unified guest + user view) ────────────────────────
// Template: templates/audience.html
// SQL: guest_codes.sql → ListAllAudience
// Depends on: formatIDR, containsCI, resolveIPLocation (helpers.go)

type audienceRow struct {
	Email          string
	UserType       string
	GuestCode      string
	ReferralSource string
	TotalLogins    int64
	AmountPaid     string
	Location       string
	LastActive     string
}

type audienceStats struct {
	Total       int
	Guests      int
	Subscribers int
	Revenue     string
}

func (h *AdminHandler) Audience(c *gin.Context) {
	ctx := c.Request.Context()
	typeFilter := c.DefaultQuery("type", "")
	query := c.DefaultQuery("q", "")
	partial := c.DefaultQuery("partial", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	perPage := int32(30)
	offset := int32(page-1) * perPage

	all, err := h.queries.ListAllAudience(ctx, repository.ListAllAudienceParams{
		Limit: 500, Offset: 0,
	})
	if err != nil {
		log.Printf("[Audience] query error: %v", err)
	}

	// Filter by type and search
	var filtered []repository.ListAllAudienceRow
	for _, a := range all {
		if typeFilter != "" && a.UserType != typeFilter {
			continue
		}
		if query != "" {
			emailMatch := containsCI(a.Email, query)
			refMatch := containsCI(a.ReferralSource, query)
			codeMatch := containsCI(a.GuestCode, query)
			if !emailMatch && !refMatch && !codeMatch {
				continue
			}
		}
		filtered = append(filtered, a)
	}

	// Stats
	stats := audienceStats{Total: len(filtered)}
	var totalRevenue int64
	for _, a := range filtered {
		switch a.UserType {
		case "guest":
			stats.Guests++
		case "subscriber":
			stats.Subscribers++
		}
		totalRevenue += a.AmountPaid
	}
	stats.Revenue = formatIDR(totalRevenue)

	// Paginate
	total := len(filtered)
	start := int(offset)
	if start > total {
		start = total
	}
	end := start + int(perPage)
	if end > total {
		end = total
	}
	paged := filtered[start:end]

	// Resolve IPs to locations
	ipCache := make(map[string]string)
	var rows []audienceRow
	for _, a := range paged {
		loc := ""
		ipStr := fmt.Sprintf("%v", a.IpAddress)
		if ipStr == "<nil>" {
			ipStr = ""
		}
		if ipStr != "" {
			if cached, ok := ipCache[ipStr]; ok {
				loc = cached
			} else {
				loc = resolveIPLocation(ipStr)
				ipCache[ipStr] = loc
			}
		}

		rows = append(rows, audienceRow{
			Email:          a.Email,
			UserType:       a.UserType,
			GuestCode:      a.GuestCode,
			ReferralSource: a.ReferralSource,
			TotalLogins:    a.TotalLogins,
			AmountPaid:     formatIDR(a.AmountPaid),
			Location:       loc,
			LastActive:     a.LastActive.Time.Format("02 Jan 15:04"),
		})
	}

	shown := end
	hasMore := end < total

	data := gin.H{
		"Title":      "Audience",
		"active":     "audience",
		"Audience":   rows,
		"Stats":      stats,
		"Query":      query,
		"TypeFilter": typeFilter,
		"Page":       page,
		"Total":      total,
		"Shown":      shown,
		"HasMore":    hasMore,
		"NextPage":   page + 1,
	}

	// Partial mode: return just the rows (for Load More)
	if partial == "1" {
		tmpl := template.Must(template.ParseFS(h.fs, "templates/audience.html"))
		c.Header("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(c.Writer, "audience-rows", data); err != nil {
			log.Printf("partial template error: %v", err)
			c.String(500, "Template error: "+err.Error())
		}
		return
	}

	h.render(c, "audience", data)
}
