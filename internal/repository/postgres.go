package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"validafianza-server/internal/models"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(dbURL string) (*PostgresRepo, error) {
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse db URL: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 5

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("unable to ping db: %w", err)
	}

	return &PostgresRepo{pool: pool}, nil
}

func (r *PostgresRepo) Close() {
	r.pool.Close()
}

func (r *PostgresRepo) GetProviderByKey(ctx context.Context, key string) (*models.ValidationProvider, error) {
	query := `
		SELECT id, provider_type, provider_key, scraper_type, config, is_active
		FROM providers
		WHERE provider_key = $1 AND is_active = true
	`

	var provider models.ValidationProvider
	err := r.pool.QueryRow(ctx, query, key).Scan(
		&provider.ID,
		&provider.ProviderType,
		&provider.ProviderKey,
		&provider.ScraperType,
		&provider.Config,
		&provider.IsActive,
	)

	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

	return &provider, nil
}

func (r *PostgresRepo) Pool() *pgxpool.Pool {
	return r.pool
}
