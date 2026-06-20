package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

func Connect() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://certpulse_user:certpulse_password@localhost:5432/certpulse?sslmode=disable"
	}

	var err error
	var config *pgxpool.Config
	config, err = pgxpool.ParseConfig(connStr)
	if err != nil {
		log.Fatalf("Unable to parse DATABASE_URL: %v\n", err)
	}

	// Configure pool parameters
	config.MaxConns = 20
	config.MinConns = 2
	config.MaxConnIdleTime = 30 * time.Minute

	// Establish connection pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	Pool, err = pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	// Verify connection
	err = Pool.Ping(ctx)
	if err != nil {
		log.Fatalf("Database ping failed: %v\n", err)
	}

	fmt.Println("Successfully connected to PostgreSQL database pool!")
}

func Close() {
	if Pool != nil {
		Pool.Close()
		fmt.Println("PostgreSQL database pool connection closed.")
	}
}
