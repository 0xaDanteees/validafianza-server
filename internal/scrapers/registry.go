package scrapers

import (
	"fmt"
	"sync"

	"validafianza-server/internal/scrapers/polizas"
)

var (
	registry = make(map[string]Scraper)
	mu		sync.RWMutex
)

func init() {
	Register(polizas.NewMapfreScraper())
	// here we add as many as we have
}

func Register(scraper Scraper) {
	mu.Lock()
	defer mu.Unlock()
	registry[scraper.Name()] = scraper
}

func Get(name string) (Scraper, error) {
	mu.RLock()
	defer mu.RUnlock()

	scraper, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("scraper not found: %s", name)
	}
	return scraper, nil
}

func List() []string {
	mu.RLock()
	defer mu.RUnlock()

	names:= make([]string, 0, len(registry))
	for name := range registry { 
		names = append(names, name)
	}
	return names
}