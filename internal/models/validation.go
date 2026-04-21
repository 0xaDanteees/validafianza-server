package models

import "time"

// ValidationRequest is the payload for POST /api/v1/polizas/validate
type ValidationRequest struct {
	// Afianzadora key — must match a registered provider
	ProviderKey string `json:"provider_key" example:"MAPFRE"`
	// Policy number / línea de validación
	LookupKey string `json:"lookup_key" example:"0032400003909000000491774"`
	// Skip cache and force a fresh scrape
	ForceRefresh bool `json:"force_refresh" example:"false"`
}

// ValidationResponse is the result of a policy validation
type ValidationResponse struct {
	// True when the policy is valid and active
	Valid bool `json:"valido" example:"true"`
	// True when the policy could not be validated
	Invalid bool `json:"no_valido" example:"false"`
	// Human-readable result message
	Message string `json:"mensaje,omitempty" example:"Póliza validada exitosamente en MAPFRE"`
	// Provider-specific structured data
	Data map[string]interface{} `json:"data,omitempty"`
	// Raw HTML fragments extracted from the provider page
	HTMLExtract []string `json:"html_extraido,omitempty"`
	// When the result was served from cache
	CachedAt *time.Time `json:"cached_at,omitempty"`
	// When the scrape was performed
	ScrapedAt *time.Time `json:"scraped_at,omitempty"`
}

// ErrorResponse wraps API error messages
type ErrorResponse struct {
	Error string `json:"error" example:"provider not found: UNKNOWN"`
}

// HealthResponse is the response for GET /health
type HealthResponse struct {
	Status  string `json:"status" example:"ok"`
	Service string `json:"service" example:"validafianza-api"`
}

// ProvidersResponse is the response for GET /api/v1/polizas/providers
type ProvidersResponse struct {
	Providers []string `json:"providers" example:"MAPFRE,BERKLEY_MEXICO_FIANZAS,SOFIMEX"`
}

// ValidationProvider is the DB record for a registered afianzadora
type ValidationProvider struct {
	ID           string                 `db:"id" json:"id" example:"a1b2c3d4-e5f6-7890-abcd-ef1234567890"`
	ProviderType string                 `db:"provider_type" json:"provider_type" example:"poliza"`
	ProviderKey  string                 `db:"provider_key" json:"provider_key" example:"MAPFRE"`
	ScraperType  string                 `db:"scraper_type" json:"scraper_type" example:"native"`
	Config       map[string]interface{} `db:"config" json:"config"`
	Metadata     map[string]string      `db:"metadata" json:"metadata"`
	IsActive     bool                   `db:"is_active" json:"is_active" example:"true"`
}

// ScrapeRequest is the internal request passed to each scraper
type ScrapeRequest struct {
	ProviderKey string
	LookupKey   string
	Config      map[string]interface{}
}

// ScrapeResult is the raw output from a scraper before caching
type ScrapeResult struct {
	Valid       bool
	Invalid     bool
	Message     string
	Data        map[string]interface{}
	HTMLExtract []string
	Error       error
}
