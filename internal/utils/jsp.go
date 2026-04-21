package utils

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

func FetchJSPPage(ctx context.Context, baseURL, lookupKey string) (*html.Node, error) {
	target := baseURL + "?txtLineaCaptura=" + url.QueryEscape(lookupKey)

	req, err := http.NewRequestWithContext(ctx, "GET", target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "es-419,es;q=0.9")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return html.Parse(strings.NewReader(string(body)))
}
