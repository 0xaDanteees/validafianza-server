package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

const initialURL = "https://validacionfianzas.mapfre.com.mx/ValidaMapfre/Sitios/Default.aspx"

func getRequestMapfre(lineaValidacion string) (*html.Node, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("Error creando cookie jar: %v", err)
	}

	client := &http.Client{Jar: jar}

	headers := map[string]string{
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
		"Accept-Language": "en-US,en;q=0.9",
		"Referer":         "https://validafianza.berkleymex.com",
		"Connection":      "keep-alive",
		"Upgrade-Insecure-Requests": "1",
	}

	getReq, err := http.NewRequest("GET", initialURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creando GET request, %v", err)
	}

	for key, value := range headers {
		getReq.Header.Set(key, value)
	}

	getResp, err := client.Do(getReq)
	if err != nil {
		return nil, fmt.Errorf("error haciendo GET request, %v", err)
	}
	defer getResp.Body.Close()


	body, err := io.ReadAll(getResp.Body)
	if err != nil {
		return nil, fmt.Errorf("error leyendo respuesta, %v", err)
	}

	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("error parseando HTML, %v", err)
	}

	formData := url.Values{}

	extractHiddenInputs(doc, formData)

	formData.Set("ctl00$maincontent$ap1_content$lvCompleto", lineaValidacion)
	formData.Set("ctl00$maincontent$bValida", "Validar")

	postReq, err := http.NewRequest("POST", initialURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("error creando POST request, %v", err)
	}

	for key, value := range headers {
		postReq.Header.Set(key, value)
	}
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	postResp, err := client.Do(postReq)
	if err != nil {
		return nil, fmt.Errorf("error haciendo POST request, %v", err)
	}
	defer postResp.Body.Close()
	postBody, err := io.ReadAll(postResp.Body)
	if err != nil {
		return nil, fmt.Errorf("error leyendo respuesta, %v", err)
	}
	
	resultDoc, err := html.Parse(strings.NewReader(string(postBody)))
	if err != nil {
		return nil, fmt.Errorf("error parseando HTML, %v", err)
	}
	
	return resultDoc, nil
}


func extractHiddenInputs(node *html.Node, data url.Values) {
	if node.Type == html.ElementNode && node.Data == "input" {
		attrs := map[string]string{}
		for _, attr := range node.Attr {
			attrs[attr.Key] = attr.Val
		}
		t := attrs["type"]
		if t == "hidden"  || t == "submit"{
			name := attrs["name"]
			if name != "" {
				data.Set(name, attrs["value"])
			}
		}
	}

	for c := node.FirstChild; c != nil; c = c.NextSibling {
		extractHiddenInputs(c, data)
	}
}


func isValidMapfre(doc *html.Node) int {
	pattern := regexp.MustCompile(`(?i)no se pudo validar`)
	if searchText(doc, pattern) {
		return 404
	}
	return 200
}


func searchText(node *html.Node, pattern *regexp.Regexp) bool {
	if node.Type == html.TextNode {
		if pattern.MatchString(node.Data) {
			return true
		}
	}
	
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if searchText(c, pattern) {
			return true
		}
	}
	
	return false
}

func main() {
	lineaValidacion := "0032400003909000000491774"

	doc, err := getRequestMapfre(lineaValidacion)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	status := isValidMapfre(doc)
	fmt.Printf("Status: %d\n", status)
	fmt.Printf("Document: %v\n", doc)
}
