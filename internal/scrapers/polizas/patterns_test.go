package polizas

import (
	"regexp"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// htmlDoc is a test helper that parses an HTML string.
func htmlDoc(t *testing.T, src string) *html.Node {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("failed to parse HTML: %v", err)
	}
	return doc
}

// validationPatternCase is a single row in the table-driven suite.
type validationPatternCase struct {
	name    string
	html    string
	wantHit bool // whether the pattern should match
}

func runPatternCases(t *testing.T, pattern *regexp.Regexp, cases []validationPatternCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := searchText(htmlDoc(t, tc.html), pattern)
			if got != tc.wantHit {
				t.Errorf("searchText() = %v, want %v", got, tc.wantHit)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// BERKLEY — "VIGENTE"
// ---------------------------------------------------------------------------

func TestBerkleyPattern(t *testing.T) {
	pattern := regexp.MustCompile(`(?i)VIGENTE`)
	runPatternCases(t, pattern, []validationPatternCase{
		{
			name:    "valid: exact match",
			html:    `<html><body><span>Estado: VIGENTE</span></body></html>`,
			wantHit: true,
		},
		{
			name:    "valid: lowercase",
			html:    `<html><body><p>vigente</p></body></html>`,
			wantHit: true,
		},
		{
			name:    "invalid: not found",
			html:    `<html><body><p>No encontrado</p></body></html>`,
			wantHit: false,
		},
		{
			name:    "invalid: empty page",
			html:    `<html><body></body></html>`,
			wantHit: false,
		},
	})
}

// ---------------------------------------------------------------------------
// SOFIMEX — "no se pudo validar" (valid = pattern NOT found)
// ---------------------------------------------------------------------------

func TestSofimexPattern(t *testing.T) {
	pattern := regexp.MustCompile(`(?i)no se pudo validar`)
	runPatternCases(t, pattern, []validationPatternCase{
		{
			name:    "invalid policy: error message present",
			html:    `<html><body><span>No se pudo validar la póliza</span></body></html>`,
			wantHit: true,
		},
		{
			name:    "valid policy: no error message",
			html:    `<html><body><p>Póliza encontrada</p></body></html>`,
			wantHit: false,
		},
	})
}

// ---------------------------------------------------------------------------
// DORAMA — "Código de validación:"
// ---------------------------------------------------------------------------

func TestDoramaPattern(t *testing.T) {
	pattern := regexp.MustCompile(`(?i)Código de validación:`)
	runPatternCases(t, pattern, []validationPatternCase{
		{
			name:    "valid: código present",
			html:    `<html><body><p>Código de validación: ABC123</p></body></html>`,
			wantHit: true,
		},
		{
			name:    "valid: different casing",
			html:    `<html><body><p>CÓDIGO DE VALIDACIÓN: XYZ</p></body></html>`,
			wantHit: true,
		},
		{
			name:    "invalid: no code",
			html:    `<html><body><p>Sin resultado</p></body></html>`,
			wantHit: false,
		},
	})
}

// ---------------------------------------------------------------------------
// TOKIO MARINE — "ANOTE ESTE NÚMERO DE VALIDACIÓN"
// ---------------------------------------------------------------------------

func TestTokioPattern(t *testing.T) {
	pattern := regexp.MustCompile(`(?i)ANOTE ESTE NÚMERO DE VALIDACIÓN`)
	runPatternCases(t, pattern, []validationPatternCase{
		{
			name:    "valid: validation number present",
			html:    `<html><body><strong>ANOTE ESTE NÚMERO DE VALIDACIÓN: 999888</strong></body></html>`,
			wantHit: true,
		},
		{
			name:    "invalid: not found",
			html:    `<html><body><p>Póliza no encontrada</p></body></html>`,
			wantHit: false,
		},
	})
}

// ---------------------------------------------------------------------------
// ATLAS — "La fianza fue validada exitosamente"
// ---------------------------------------------------------------------------

func TestAtlasPattern(t *testing.T) {
	pattern := regexp.MustCompile(`(?i)La fianza fue validada exitosamente`)
	runPatternCases(t, pattern, []validationPatternCase{
		{
			name:    "valid: success message present",
			html:    `<html><body><span>La fianza fue validada exitosamente.</span></body></html>`,
			wantHit: true,
		},
		{
			name:    "invalid: error message",
			html:    `<html><body><span>ERROR validación ante el SAT: no devuelve ningún documento</span></body></html>`,
			wantHit: false,
		},
		{
			name:    "invalid: partial match not enough",
			html:    `<html><body><span>La fianza fue procesada</span></body></html>`,
			wantHit: false,
		},
	})
}

// ---------------------------------------------------------------------------
// MAPFRE — "no se pudo validar" (valid = pattern NOT found)
// ---------------------------------------------------------------------------

func TestMapfreErrorPattern(t *testing.T) {
	pattern := regexp.MustCompile(`(?i)no se pudo validar`)
	runPatternCases(t, pattern, []validationPatternCase{
		{
			name:    "error message found",
			html:    `<html><body><div class="error">No se pudo validar la fianza</div></body></html>`,
			wantHit: true,
		},
		{
			name:    "valid policy: no error",
			html:    `<html><body><table class="mainTable"><tr><td>VIGENTE</td></tr></table></body></html>`,
			wantHit: false,
		},
	})
}

// ---------------------------------------------------------------------------
// ASERTA / INSURGENTES (JSP) — "no existe" (valid = pattern NOT found)
// ---------------------------------------------------------------------------

func TestJSPPattern(t *testing.T) {
	pattern := regexp.MustCompile(`(?i)no existe`)
	runPatternCases(t, pattern, []validationPatternCase{
		{
			name:    "policy not found",
			html:    `<html><body><p>La fianza no existe en nuestros registros</p></body></html>`,
			wantHit: true,
		},
		{
			name:    "valid policy: data returned",
			html:    `<html><body><table><tr><td>Afianzado: EMPRESA SA</td></tr></table></body></html>`,
			wantHit: false,
		},
		{
			name:    "empty page treated as valid",
			html:    `<html><body></body></html>`,
			wantHit: false,
		},
	})
}

// ---------------------------------------------------------------------------
// searchText — edge cases
// ---------------------------------------------------------------------------

func TestSearchTextEdgeCases(t *testing.T) {
	pattern := regexp.MustCompile(`TARGET`)

	t.Run("deeply nested node", func(t *testing.T) {
		doc := htmlDoc(t, `<html><body><div><div><span><p>TARGET</p></span></div></div></body></html>`)
		if !searchText(doc, pattern) {
			t.Error("expected match in deeply nested node")
		}
	})

	t.Run("text split across siblings", func(t *testing.T) {
		// Each sibling text node is checked independently — no cross-node match
		doc := htmlDoc(t, `<html><body><span>TAR</span><span>GET</span></body></html>`)
		if searchText(doc, pattern) {
			t.Error("pattern should not match across sibling text nodes")
		}
	})

	t.Run("nil-safe on empty document", func(t *testing.T) {
		doc := htmlDoc(t, ``)
		// should not panic
		searchText(doc, pattern)
	})
}
