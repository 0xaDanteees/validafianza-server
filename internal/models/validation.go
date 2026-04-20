package models

import (
	"context"
	"time"
)

type ValidationProvider struct {
	ID				int32					`db:"id" json:"id"`
	ProviderType	string 					`db:"provider_type" json:"provider_type"`
	ProviderKey 	string 					`db:"provider_key" json:"provider_type"`
	ScraperType 	string		 			`db:"scraper_type" json:"scraper_type"`
	Config 			map[string]interface{} 	`db:"config" json:"config"`
	Metadata 		map[string]string 		`db:"metadata" json:;"metadata"`
	IsActive 		bool 					`db:"is_active" json:"is_active"`
}

type ValidationRequest struct {
	ProviderKey		string					`json:"provider_key"` binding:"required"``
	LookupKey		string					`json:"lookup_key" binding:"required"`
	ForceRefresh	bool					`json:"force_refresh"`
}

type ValidationResponse struct {
	valid			bool					`json:"valido"`
	Invalid			bool					`json:"no_valido"`
	Message			string					`json:"mesaje,omitempty"`
	Data			map[string]interface{}	`json:"data,omitempty"`
	HTMLExtract		[]string				`json:"html_extraido,omitempty"`
	CachedAt		*time.Time				`json:"cached_at,omitempty"`
	ScrapedAt		*time.Time				`json:"scraped_at,omitempty"`
}

type ScrapeRequest struct {
	ProviderKey		string
	LookupKey		string
	Config			map[string]interface{}
}

type ScrapeResult struct {
	Valid			bool
	Invalid			bool
	Message			string
	Data			map[string]interface{}
	HTMLExtract		[]string
	Error			error
}