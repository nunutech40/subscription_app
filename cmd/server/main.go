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
	"github.com/joho/godotenv"
	"github.com/nununugraha/sains-api/internal/config"
	"github.com/nununugraha/sains-api/internal/database"
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

	// Set Gin mode
	gin.SetMode(cfg.GinMode)

	// Create router
	r := gin.Default()

	// ── Health check ─────────────────────────────────────────────────
	r.GET("/health", func(c *gin.Context) {
		// Ping DB to verify connection is alive
		dbStatus := "ok"
		if err := dbPool.Ping(c.Request.Context()); err != nil {
			dbStatus = "error: " + err.Error()
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "sains-api",
			"db":      dbStatus,
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// ── API routes ───────────────────────────────────────────────────
	api := r.Group("/api")
	{
		api.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "pong"})
		})
	}

	// ── Start server with graceful shutdown ───────────────────────────
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

	// Wait for interrupt signal
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
