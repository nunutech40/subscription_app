package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nununugraha/sains-api/internal/service"
)

// AuthMiddleware validates JWT tokens from Authorization header or cookie.
func AuthMiddleware(tokenService *service.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractToken(c)
		if tokenString == "" {
			abortWithError(c, http.StatusUnauthorized, "TOKEN_INVALID", "Token tidak ditemukan. Silakan login.")
			return
		}

		claims, err := tokenService.ValidateToken(tokenString)
		if err != nil {
			code := "TOKEN_INVALID"
			msg := "Token tidak valid"
			if err == service.ErrExpiredToken {
				code = "TOKEN_EXPIRED"
				msg = "Token sudah expired. Silakan refresh atau login ulang."
			}
			abortWithError(c, http.StatusUnauthorized, code, msg)
			return
		}

		// Set user info in context for downstream handlers
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// AdminMiddleware checks that the authenticated user has admin role.
// Must be used AFTER AuthMiddleware.
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role.(string) != "admin" {
			abortWithError(c, http.StatusForbidden, "FORBIDDEN", "Akses ditolak. Hanya admin yang bisa mengakses.")
			return
		}
		c.Next()
	}
}

// extractToken gets JWT from Authorization header (Bearer) or cookie.
func extractToken(c *gin.Context) string {
	// 1. Try Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
			return parts[1]
		}
	}

	// 2. Try cookie
	if token, err := c.Cookie("access_token"); err == nil {
		return token
	}

	return ""
}

// abortWithError sends a consistent error response from middleware.
// Uses the same { error: { code, message } } format as handler/response.go.
func abortWithError(c *gin.Context, status int, code string, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}
