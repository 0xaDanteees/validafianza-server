package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"validafianza-server/internal/models"
)

type CacheRepo struct {
	pool *pgxpool.Pool
}

func NewCacheRepo(pool *pgxpool.Pool) *CacheRepo {
	return &CacheRepo{pool: pool}
}

func (r *CacheRepo) Get(ctx context.Context, lookupHash string) (*models.ValidationResponse, error) {
	query := `
		UPDATE validation_cache
		SET access_count = access_count + 1
		WHERE lookup_hash = $1 AND expires > NOW()
		RETURNING result, scraped_at
	`

	var resultJSON []byte
	var scrapedAt time.Time

	err := r.pool.QueryRow(ctx, query, lookupHash).Scan(&resultJSON, &scrapedAt)

	if err != nil {
		return nil, err
	}

	var response models.ValidationResponse
	if err := json.Unmarshal(resultJSON, &response); err != nil {
		return nil, err
	}

	response.CachedAt = &scrapedAt
	return &response, nil
}


func (r *CacheRepo) Set(ctx context.Context, lookupHash string, providerID int32, lookupKey string, reponse *models.ValidationResponse, ttl time.Duration) error {
	resultJSON, err := json.Marshal(response)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO validation_cache (lookup_hash, provider_id, lookup_key, result, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (lookup_hash) DO UPDATE
		SET 
			esult = EXCLUDED.result, 
			access_count = validation_cache.access_count + 1,
			scraped_at = NOW()
			expires_at = EXCLUDED.expires_at
	`

	expiresAt := time.Now().add(ttl)
	_, err = r.pool.Exec(ctx, query, lookupHash, providerID, lookupKey, resultJSON, expiresAt)
	return err
}