package admin

import (
	"html/template"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nununugraha/sains-api/internal/service"
)

// ── Admin Auth ───────────────────────────────────────────────────
// Template: templates/login.html
// Depends on: uuidStr (helpers.go), AuthService, TokenService

// AdminAuthMiddleware checks for admin_token cookie with valid admin JWT.
func (h *AdminHandler) AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr, err := c.Cookie("admin_token")
		if err != nil || tokenStr == "" {
			c.Redirect(http.StatusFound, "/admin/login")
			c.Abort()
			return
		}

		claims, err := h.tokenService.ValidateToken(tokenStr)
		if err != nil {
			// Clear invalid cookie
			c.SetCookie("admin_token", "", -1, "/admin", "", false, true)
			c.Redirect(http.StatusFound, "/admin/login")
			c.Abort()
			return
		}

		if claims.Role != "admin" {
			c.SetCookie("admin_token", "", -1, "/admin", "", false, true)
			c.Redirect(http.StatusFound, "/admin/login")
			c.Abort()
			return
		}

		// Set admin info in context
		c.Set("admin_id", claims.UserID)
		c.Set("admin_email", claims.Email)
		c.Next()
	}
}

// LoginPage serves GET /admin/login
func (h *AdminHandler) LoginPage(c *gin.Context) {
	// If already authenticated, redirect to dashboard
	if tokenStr, err := c.Cookie("admin_token"); err == nil && tokenStr != "" {
		if claims, err := h.tokenService.ValidateToken(tokenStr); err == nil && claims.Role == "admin" {
			c.Redirect(http.StatusFound, "/admin/")
			return
		}
	}

	tmpl := template.Must(template.ParseFS(h.fs, "templates/login.html"))
	c.Header("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(c.Writer, gin.H{"Error": "", "Email": ""})
}

// LoginPost handles POST /admin/login
func (h *AdminHandler) LoginPost(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")

	result, err := h.authService.Login(c.Request.Context(), service.LoginInput{
		Email:    email,
		Password: password,
	})
	if err != nil {
		tmpl := template.Must(template.ParseFS(h.fs, "templates/login.html"))
		c.Header("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(c.Writer, gin.H{"Error": "Email atau password salah.", "Email": email})
		return
	}

	// Check admin role
	if result.User.Role.String != "admin" {
		tmpl := template.Must(template.ParseFS(h.fs, "templates/login.html"))
		c.Header("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(c.Writer, gin.H{"Error": "Hanya akun admin yang bisa mengakses dashboard.", "Email": email})
		return
	}

	// Generate JWT for admin session
	token, err := h.tokenService.GenerateAccessToken(
		uuidStr(result.User.ID),
		result.User.Email,
		result.User.Role.String,
	)
	if err != nil {
		log.Printf("admin token error: %v", err)
		tmpl := template.Must(template.ParseFS(h.fs, "templates/login.html"))
		c.Header("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(c.Writer, gin.H{"Error": "Gagal membuat sesi. Coba lagi.", "Email": email})
		return
	}

	// Set httpOnly cookie (24 hours)
	c.SetCookie("admin_token", token, 24*60*60, "/admin", "", false, true)
	c.Redirect(http.StatusFound, "/admin/")
}

// Logout handles GET /admin/logout
func (h *AdminHandler) Logout(c *gin.Context) {
	c.SetCookie("admin_token", "", -1, "/admin", "", false, true)
	c.Redirect(http.StatusFound, "/admin/login")
}
