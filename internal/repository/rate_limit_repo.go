package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RateLimitRepo struct {
	pool *pgxpool.Pool
}

func NewRateLimitRepo(pool *pgxpool.Pool) *RateLimitRepo {
	return &RateLimitRepo{pool: pool}
}

func (r *RateLimitRepo) Allow(ctx context.Context, key string, limit, windowSeconds int) (bool, error) {
	var allowed bool
	err := r.pool.QueryRow(ctx,
		`SELECT check_rate_limit($1, $2, $3)`,
		key, limit, windowSeconds,
	).Scan(&allowed)
	if err != nil {
		return true, err
	}
	return allowed, nil
}
