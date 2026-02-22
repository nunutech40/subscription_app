package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, "UPDATE users SET role = 'admin', is_active = TRUE WHERE email = 'test@sains.id'")
	if err != nil {
		panic(err)
	}
	fmt.Println("✅ User test@sains.id is now admin + active")
}
