package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool creates a new pgx connection pool to PostgreSQL.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse DATABASE_URL: %w", err)
	}

	// Disable prepared statement cache for PgBouncer (Supabase pooler) compatibility
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	// Connection pool settings
	config.MaxConns = 10                   // max connections (Supabase free tier limit ~15)
	config.MinConns = 2                    // keep 2 idle connections ready
	config.MaxConnLifetime = 1 * time.Hour // recycle connections every hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	log.Printf("✅ Connected to PostgreSQL (pool: min=%d, max=%d)", config.MinConns, config.MaxConns)

	return pool, nil
}

// Close gracefully closes the connection pool.
func Close(pool *pgxpool.Pool) {
	if pool != nil {
		pool.Close()
		log.Println("📦 Database connection pool closed")
	}
}
