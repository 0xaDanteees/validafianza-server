package ScrapeResult

import (
	"context"
	"validafianza-server/internal/models"
)

type Scraper interface {
	Scrape(ctx context.Context, req *models.ScrapeRequest)
	(*models.ScrapeResult, error)

	Name() string
	Type() string
}

type BaseScraper struct {
	name		string
	scraperType	string
}

func NewBaseScraper(name, scraperType string)
*BaseScraper {
	return &BaseScraper{
		name: name,
		scraperType: scraperType,
	}
}

func (b *BaseScraper) Name() string {
	return b.name
}

func (b *BaseScraper) Type() string {
	return b.scraperType
}