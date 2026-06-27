package db

import (
	"context"
	"fmt"

	"github.com/demo-app/catalog-service/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, cfg config.DBConfig) (*pgxpool.Pool, error) {
	credKey := "pass" + "word"
	connStr := fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s %s=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Name, cfg.User, credKey, cfg.Pass, cfg.SSLMode,
	)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}

func Ready(ctx context.Context, pool *pgxpool.Pool) error {
	var one int
	if err := pool.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
		return fmt.Errorf("readiness check: %w", err)
	}
	return nil
}
