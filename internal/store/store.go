package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Product struct {
	ID         int       `json:"id"`
	Name       string    `json:"name"`
	SKU        string    `json:"sku"`
	PriceCents int       `json:"price_cents"`
	CreatedAt  time.Time `json:"created_at"`
}

type ProductInput struct {
	Name       string `json:"name"`
	SKU        string `json:"sku"`
	PriceCents int    `json:"price_cents"`
}

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) ListProducts(ctx context.Context) ([]Product, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, sku, price_cents, created_at
		FROM products
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("query products: %w", err)
	}
	defer rows.Close()

	products := make([]Product, 0)
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.SKU, &p.PriceCents, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate products: %w", err)
	}

	return products, nil
}

func (s *Store) GetProduct(ctx context.Context, id int) (Product, error) {
	var p Product
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, sku, price_cents, created_at
		FROM products
		WHERE id = $1
	`, id).Scan(&p.ID, &p.Name, &p.SKU, &p.PriceCents, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Product{}, ErrNotFound
		}
		return Product{}, fmt.Errorf("query product: %w", err)
	}
	return p, nil
}

func (s *Store) CreateProduct(ctx context.Context, in ProductInput) (Product, error) {
	var p Product
	err := s.pool.QueryRow(ctx, `
		INSERT INTO products (name, sku, price_cents)
		VALUES ($1, $2, $3)
		RETURNING id, name, sku, price_cents, created_at
	`, in.Name, in.SKU, in.PriceCents).Scan(&p.ID, &p.Name, &p.SKU, &p.PriceCents, &p.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return Product{}, ErrDuplicateSKU
		}
		return Product{}, fmt.Errorf("insert product: %w", err)
	}
	return p, nil
}

func (s *Store) UpdateProduct(ctx context.Context, id int, in ProductInput) (Product, error) {
	var p Product
	err := s.pool.QueryRow(ctx, `
		UPDATE products
		SET name = $2, sku = $3, price_cents = $4
		WHERE id = $1
		RETURNING id, name, sku, price_cents, created_at
	`, id, in.Name, in.SKU, in.PriceCents).Scan(&p.ID, &p.Name, &p.SKU, &p.PriceCents, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Product{}, ErrNotFound
		}
		if isUniqueViolation(err) {
			return Product{}, ErrDuplicateSKU
		}
		return Product{}, fmt.Errorf("update product: %w", err)
	}
	return p, nil
}

func (s *Store) DeleteProduct(ctx context.Context, id int) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

var (
	ErrNotFound     = errors.New("product not found")
	ErrDuplicateSKU = errors.New("sku already exists")
)
