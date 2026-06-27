package db

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/demo-app/catalog-service/internal/config"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Pool struct {
	mu   sync.RWMutex
	pool *pgxpool.Pool
}

func NewPool(ctx context.Context, cfg config.DBConfig) (*Pool, error) {
	p, err := openPool(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &Pool{pool: p}, nil
}

func openPool(ctx context.Context, cfg config.DBConfig) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}

func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pool != nil {
		p.pool.Close()
		p.pool = nil
	}
}

func (p *Pool) Underlying() *pgxpool.Pool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.pool
}

func (p *Pool) Reload(ctx context.Context, cfg config.DBConfig) error {
	next, err := openPool(ctx, cfg)
	if err != nil {
		return err
	}

	p.mu.Lock()
	old := p.pool
	p.pool = next
	p.mu.Unlock()

	if old != nil {
		old.Close()
	}
	return nil
}

func (p *Pool) Ready(ctx context.Context) error {
	p.mu.RLock()
	pool := p.pool
	p.mu.RUnlock()

	var one int
	if err := pool.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
		return fmt.Errorf("readiness check: %w", err)
	}
	return nil
}

func IsAuthError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "28P01"
}
