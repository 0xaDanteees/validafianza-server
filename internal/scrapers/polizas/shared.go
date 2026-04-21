package polizas

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

func searchText(node *html.Node, pattern *regexp.Regexp) bool {
	if node.Type == html.TextNode && pattern.MatchString(node.Data) {
		return true
	}
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if searchText(c, pattern) {
			return true
		}
	}
	return false
}

func renderNode(n *html.Node) string {
	var buf strings.Builder
	html.Render(&buf, n)
	return buf.String()
}
