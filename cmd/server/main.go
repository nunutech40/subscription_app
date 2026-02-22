package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/nununugraha/sains-api/internal/config"
	"github.com/nununugraha/sains-api/internal/database"
	"github.com/nununugraha/sains-api/internal/handler"
	"github.com/nununugraha/sains-api/internal/middleware"
	"github.com/nununugraha/sains-api/internal/repository"
	"github.com/nununugraha/sains-api/internal/service"
)

func main() {
	// Load .env file (ignore error in production)
	_ = godotenv.Load()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// ── Database ──────────────────────────────────────────────────────
	ctx := context.Background()
	dbPool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close(dbPool)

	// ── Dependencies ─────────────────────────────────────────────────
	queries := repository.New(dbPool)

	// Parse JWT expiry duration
	jwtExpiry, err := time.ParseDuration(cfg.JWTExpiry)
	if err != nil {
		jwtExpiry = 1 * time.Hour
	}

	tokenService := service.NewTokenService(cfg.JWTSecret, jwtExpiry)
	authService := service.NewAuthService(queries)
	xenditService := service.NewXenditService(cfg.XenditAPIKey, cfg.XenditBaseURL)
	emailService := service.NewEmailService(cfg.ResendAPIKey, cfg.FromEmail)

	// ── Handlers ─────────────────────────────────────────────────────
	authHandler := handler.NewAuthHandler(authService, tokenService, queries, cfg.RefreshTokenExpiryDays)
	planHandler := handler.NewPlanHandler(queries)
	subHandler := handler.NewSubscriptionHandler(queries, xenditService, emailService, cfg.XenditWebhookToken, cfg.FrontendURL)

	// ── Router ────────────────────────────────────────────────────────
	gin.SetMode(cfg.GinMode)
	r := gin.New()

	// Global middleware
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORSMiddleware(cfg.CORSOrigins))
	r.Use(gin.Logger())

	// ── Health check ─────────────────────────────────────────────────
	r.GET("/health", healthCheck(dbPool))

	// ── Public API routes ────────────────────────────────────────────
	api := r.Group("/api")
	{
		api.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "pong"})
		})

		// Auth (public)
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		// Plans (public)
		api.GET("/plans", planHandler.ListPlans)
		api.GET("/plans/:id", planHandler.GetPlan)

		// Quota status (public)
		api.GET("/quota-status", subHandler.QuotaStatus)

		// Xendit webhook (public, verified by X-Callback-Token)
		api.POST("/xendit/webhook", subHandler.XenditWebhook)
	}

	// ── Protected API routes ─────────────────────────────────────────
	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(tokenService))
	{
		// Auth (protected)
		protected.POST("/auth/logout", authHandler.Logout)
		protected.GET("/auth/me", authHandler.Me)

		// Checkout + Subscriptions
		protected.POST("/checkout", subHandler.Checkout)
		protected.GET("/subscriptions/me", subHandler.MySubscriptions)
		protected.GET("/access-check", subHandler.AccessCheck)
	}

	// ── Admin routes ─────────────────────────────────────────────────
	admin := api.Group("/admin")
	admin.Use(middleware.AuthMiddleware(tokenService))
	admin.Use(middleware.AdminMiddleware())
	{
		admin.POST("/pricing-plans", planHandler.CreatePlan)
		admin.PUT("/pricing-plans/:id", planHandler.UpdatePlan)
	}

	// ── Start server ─────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	go func() {
		log.Printf("🚀 SAINS API starting on :%s (mode: %s)", cfg.Port, cfg.GinMode)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("⏳ Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server exited cleanly")
}

func healthCheck(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		dbStatus := "ok"
		if err := pool.Ping(c.Request.Context()); err != nil {
			dbStatus = "error"
		}
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "sains-api",
			"db":      dbStatus,
			"time":    time.Now().Format(time.RFC3339),
		})
	}
}
