package services

import (
	"context"
	"time"

	"validafianza-server/internal/models"
)

type providerRepository interface {
	GetProviderByKey(ctx context.Context, key string) (*models.ValidationProvider, error)
}

type validationCache interface {
	Get(ctx context.Context, hash string) (*models.ValidationResponse, error)
	Set(ctx context.Context, hash, providerID, lookupKey string, resp *models.ValidationResponse, ttl time.Duration) error
}
