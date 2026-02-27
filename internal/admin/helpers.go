package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ── Shared Helpers ──────────────────────────────────────────────────
// These functions are used by multiple handlers.
// Edit with care — changes here affect ALL handlers that use them.
// See docs/ARCHITECTURE.md for the full dependency map.

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

func containsCI(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// resolveIPLocation uses free ip-api.com to get city+country from IP.
func resolveIPLocation(ip string) string {
	if ip == "" || ip == "127.0.0.1" || ip == "::1" {
		return "Localhost"
	}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://ip-api.com/json/" + ip + "?fields=status,city,country")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var result struct {
		Status  string `json:"status"`
		City    string `json:"city"`
		Country string `json:"country"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.Status != "success" {
		return ""
	}
	if result.City != "" {
		return result.City + ", " + result.Country
	}
	return result.Country
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
