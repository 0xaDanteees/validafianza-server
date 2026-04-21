package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"validafianza-server/internal/models"
	"validafianza-server/internal/scrapers"
)

// ---------------------------------------------------------------------------
// Fakes
// ---------------------------------------------------------------------------

type fakeRepo struct {
	provider *models.ValidationProvider
	err      error
}

func (f *fakeRepo) GetProviderByKey(_ context.Context, _ string) (*models.ValidationProvider, error) {
	return f.provider, f.err
}

type fakeCache struct {
	cached   *models.ValidationResponse
	getErr   error
	setErr   error
	setCalls int
}

func (f *fakeCache) Get(_ context.Context, _ string) (*models.ValidationResponse, error) {
	return f.cached, f.getErr
}

func (f *fakeCache) Set(_ context.Context, _, _, _ string, _ *models.ValidationResponse, _ time.Duration) error {
	f.setCalls++
	return f.setErr
}

type fakeScraper struct {
	name   string
	result *models.ScrapeResult
	err    error
}

func (f *fakeScraper) Name() string { return f.name }
func (f *fakeScraper) Type() string { return "native" }
func (f *fakeScraper) Scrape(_ context.Context, _ *models.ScrapeRequest) (*models.ScrapeResult, error) {
	return f.result, f.err
}

// activeProvider returns a minimal ValidationProvider ready for testing.
func activeProvider(key string) *models.ValidationProvider {
	return &models.ValidationProvider{
		ID:          "uuid-test-001",
		ProviderKey: key,
		IsActive:    true,
		Config:      map[string]interface{}{},
	}
}

// newService wires up a ValidationService with the given fakes.
func newService(repo providerRepository, cache validationCache) *ValidationService {
	return &ValidationService{repo: repo, cacheRepo: cache, cacheTTL: 24 * time.Hour}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestValidate_CacheHit(t *testing.T) {
	cached := &models.ValidationResponse{Valid: true, Message: "from cache"}
	svc := newService(
		&fakeRepo{provider: activeProvider("MAPFRE")},
		&fakeCache{cached: cached},
	)

	resp, err := svc.Validate(context.Background(), &models.ValidationRequest{
		ProviderKey: "MAPFRE",
		LookupKey:   "POLIZA-001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != cached {
		t.Error("expected cached response to be returned directly")
	}
}

func TestValidate_ForceRefresh_BypassesCache(t *testing.T) {
	scraper := &fakeScraper{
		name:   "MAPFRE",
		result: &models.ScrapeResult{Valid: true, Message: "fresh"},
	}
	scrapers.Register(scraper)
	t.Cleanup(func() { /* registry is global; acceptable for test suite */ })

	cache := &fakeCache{cached: &models.ValidationResponse{Valid: false, Message: "stale"}}
	svc := newService(&fakeRepo{provider: activeProvider("MAPFRE")}, cache)

	resp, err := svc.Validate(context.Background(), &models.ValidationRequest{
		ProviderKey:  "MAPFRE",
		LookupKey:    "POLIZA-001",
		ForceRefresh: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Valid {
		t.Error("expected fresh scrape result (valid=true), got cached stale result")
	}
	if cache.setCalls != 1 {
		t.Errorf("expected 1 cache write after scrape, got %d", cache.setCalls)
	}
}

func TestValidate_ProviderNotFound(t *testing.T) {
	svc := newService(
		&fakeRepo{err: errors.New("no rows")},
		&fakeCache{},
	)

	_, err := svc.Validate(context.Background(), &models.ValidationRequest{
		ProviderKey: "UNKNOWN",
		LookupKey:   "anything",
	})
	if err == nil {
		t.Fatal("expected error for unknown provider, got nil")
	}
}

func TestValidate_InactiveProvider(t *testing.T) {
	inactive := activeProvider("MAPFRE")
	inactive.IsActive = false

	svc := newService(&fakeRepo{provider: inactive}, &fakeCache{})

	resp, err := svc.Validate(context.Background(), &models.ValidationRequest{
		ProviderKey: "MAPFRE",
		LookupKey:   "POLIZA-001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Valid {
		t.Error("inactive provider should return Valid=false")
	}
}

func TestValidate_ScraperError_ReturnsError(t *testing.T) {
	errScraper := &fakeScraper{name: "MAPFRE", err: errors.New("timeout")}
	scrapers.Register(errScraper)

	svc := newService(
		&fakeRepo{provider: activeProvider("MAPFRE")},
		&fakeCache{getErr: errors.New("no cache")},
	)

	_, err := svc.Validate(context.Background(), &models.ValidationRequest{
		ProviderKey: "MAPFRE",
		LookupKey:   "POLIZA-001",
	})
	if err == nil {
		t.Fatal("expected error from scraper, got nil")
	}
}

func TestValidate_CacheWriteFailure_DoesNotAffectResponse(t *testing.T) {
	scraper := &fakeScraper{
		name:   "SOFIMEX",
		result: &models.ScrapeResult{Valid: true, Message: "ok"},
	}
	scrapers.Register(scraper)

	svc := newService(
		&fakeRepo{provider: activeProvider("SOFIMEX")},
		&fakeCache{getErr: errors.New("miss"), setErr: errors.New("write failed")},
	)

	resp, err := svc.Validate(context.Background(), &models.ValidationRequest{
		ProviderKey: "SOFIMEX",
		LookupKey:   "POLIZA-001",
	})
	if err != nil {
		t.Fatalf("cache write failure should not propagate as error, got: %v", err)
	}
	if !resp.Valid {
		t.Error("scrape result should be returned even when cache write fails")
	}
}

func TestGenerateLookupHash_Deterministic(t *testing.T) {
	svc := &ValidationService{}
	h1 := svc.generateLookupHash("uuid-001", "POLIZA-XYZ")
	h2 := svc.generateLookupHash("uuid-001", "POLIZA-XYZ")
	if h1 != h2 {
		t.Errorf("hash must be deterministic: %q != %q", h1, h2)
	}
}

func TestGenerateLookupHash_UniquePerInput(t *testing.T) {
	svc := &ValidationService{}
	cases := [][2]string{
		{"uuid-001", "POLIZA-A"},
		{"uuid-001", "POLIZA-B"},
		{"uuid-002", "POLIZA-A"},
	}
	seen := map[string]bool{}
	for _, c := range cases {
		h := svc.generateLookupHash(c[0], c[1])
		if seen[h] {
			t.Errorf("hash collision for providerID=%q lookupKey=%q", c[0], c[1])
		}
		seen[h] = true
	}
}
