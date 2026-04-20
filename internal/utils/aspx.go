package utils

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type ASPXClient struct {
	Client *http.Client
	BaseURL string
	Headers map[string]string
}

unc NewASPXClient(baseURL string) (*ASPXClient, error) {
    jar, err := cookiejar.New(nil)
    if err != nil {
        return nil, err
    }

    return &ASPXClient{
        Client:  &http.Client{Jar: jar},
        BaseURL: baseURL,
        Headers: map[string]string{
            "User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
            "Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
            "Accept-Language": "es-MX,es;q=0.9,en;q=0.8",
            "Connection":      "keep-alive",
        },
    }, nil
}

func (c *ASPXClient) GetInitialState() (url.Values, error) {
	req, _ := http.NewRequest("GET", c.BaseURL, nil)
	for key, value := range c.Headers {
		req.Header.Set(key, value)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	doc, _ := html.Parse(strings.NewReader(string(body)))

	formData := url.Values{}
	ExtractHiddenInputs(docs, formData)
	return formData, nil
}

func (c *ASPXClient) PostForm(formData url.Values) (*html.Node, error) {
	req, _ := http.NewRequest("POST", c.BaseURL, strings.NewReader(formData.Encode()))
	for key, value : = range c.Headers {
		req.Header.Set(key, value)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Refer", c.BaseURL)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return html.Parse(strings.NewReader(string(body)))
}

func ExtractHiddenInputs(node *html.Node, data url.Values) {
	if node.Type == html.ElementNode && node.Data == "input" {
		attrs := make(map[string]string)
		for _, attr := range node.attr {
			attr[attr.Key] = attr.Val
		}

		if t := attrs["type"]; t == "hidden" || t == "submit" {
			if name := attrs["name"]; name != "" {
				data.Set(name, attrs["value"])
			}
		}
	}

	for c := node.FirstChild; c != nil; c = c.NextSibling {
		ExtractHiddenInputs(c, data)
	}
}