package middleware

import (
	"net/netip"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nununugraha/sains-api/internal/repository"
)

// AuditLogger provides admin action logging.
type AuditLogger struct {
	queries *repository.Queries
}

// NewAuditLogger creates a new audit logger.
func NewAuditLogger(q *repository.Queries) *AuditLogger {
	return &AuditLogger{queries: q}
}

// Log records an admin action.
func (a *AuditLogger) Log(c *gin.Context, action, resource, resourceID, detail string) {
	adminEmail, _ := c.Get("admin_email")
	email, _ := adminEmail.(string)
	if email == "" {
		email = "unknown"
	}

	var ip *netip.Addr
	parsed, err := netip.ParseAddr(c.ClientIP())
	if err == nil {
		ip = &parsed
	}

	_ = a.queries.InsertAuditLog(c.Request.Context(), repository.InsertAuditLogParams{
		AdminEmail: email,
		Action:     action,
		Resource:   resource,
		ResourceID: pgtype.Text{String: resourceID, Valid: resourceID != ""},
		Detail:     pgtype.Text{String: detail, Valid: detail != ""},
		Ip:         ip,
	})
}
