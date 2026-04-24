package utils

import "testing"

func TestValidarRFC(t *testing.T) {
	tests := []struct {
		name string
		rfc  string
		want bool
	}{
		{"RFC persona moral válido", "SIR130227V77", true},
		{"RFC persona física válido", "MASE810219AH5", true},
		{"RFC con Ñ válido", "SEÑ850101XXX", true},
		{"RFC con & válido", "M&S850101XXX", true},
		{"RFC muy corto", "AB12345", false},
		{"RFC con caracteres inválidos", "ABC@123456XXX", false},
		{"RFC vacío", "", false},
		{"RFC con minúsculas", "sir130227v77", false},
		{"RFC solo números", "123456789012", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidarRFC(tt.rfc); got != tt.want {
				t.Errorf("ValidarRFC(%q) = %v, want %v", tt.rfc, got, tt.want)
			}
		})
	}
}
