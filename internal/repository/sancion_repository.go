package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"validafianza-server/internal/models"
)

type SancionRepository struct {
	db *pgxpool.Pool
}

func NewSancionRepository(db *pgxpool.Pool) *SancionRepository {
	return &SancionRepository{db: db}
}

func (r *SancionRepository) GetCached(ctx context.Context, rfc string) (*models.SancionCache, error) {
	query := `
		SELECT id, rfc, sancionado, detalles, created_at, expires_at
		FROM sancion_cache
		WHERE rfc = $1 AND expires_at > NOW()
	`

	var cache models.SancionCache
	err := r.db.QueryRow(ctx, query, rfc).Scan(
		&cache.ID,
		&cache.RFC,
		&cache.Sancionado,
		&cache.Detalles,
		&cache.CreatedAt,
		&cache.ExpiresAt,
	)

	if err != nil {
		return nil, err
	}

	return &cache, nil
}

func (r *SancionRepository) SaveCache(ctx context.Context, rfc string, sancionado bool, detalles string, ttlHours int) error {
	query := `
		INSERT INTO sancion_cache (id, rfc, sancionado, detalles, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (rfc) 
		DO UPDATE SET 
			sancionado = EXCLUDED.sancionado,
			detalles = EXCLUDED.detalles,
			created_at = EXCLUDED.created_at,
			expires_at = EXCLUDED.expires_at
	`

	now := time.Now()
	expiresAt := now.Add(time.Duration(ttlHours) * time.Hour)

	_, err := r.db.Exec(ctx, query,
		uuid.New(),
		rfc,
		sancionado,
		detalles,
		now,
		expiresAt,
	)

	return err
}
