package scraper

import (
	"os"
	"testing"
	"time"
)

func TestParseAnoModelo(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedAno     int
		expectedComb    int
		expectErr       bool
	}{
		{
			name:         "Válido 2022-1",
			input:        "2022-1",
			expectedAno:  2022,
			expectedComb: 1,
			expectErr:    false,
		},
		{
			name:         "Válido 32000-3 (0km)",
			input:        "32000-3",
			expectedAno:  32000,
			expectedComb: 3,
			expectErr:    false,
		},
		{
			name:      "Inválido sem hifen",
			input:     "20221",
			expectErr: true,
		},
		{
			name:      "Ano não numérico",
			input:     "abc-1",
			expectErr: true,
		},
		{
			name:      "Combustível não numérico",
			input:     "2022-x",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ano, comb, err := ParseAnoModelo(tt.input)
			if tt.expectErr {
				if err == nil {
					t.Errorf("esperava erro para input %q, mas obteve nil", tt.input)
				}
			} else {
				if err != nil {
					t.Fatalf("erro inesperado para input %q: %v", tt.input, err)
				}
				if ano != tt.expectedAno {
					t.Errorf("ano esperado %d, obtido %d", tt.expectedAno, ano)
				}
				if comb != tt.expectedComb {
					t.Errorf("combustível esperado %d, obtido %d", tt.expectedComb, comb)
				}
			}
		})
	}
}

func TestParsePreco(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
	}{
		{
			name:     "Formato padrão BRL",
			input:    "R$ 45.870,00",
			expected: 45870.00,
		},
		{
			name:     "Formato sem espaço",
			input:    "R$123.456,78",
			expected: 123456.78,
		},
		{
			name:     "Apenas números e vírgula",
			input:    "5.000,50",
			expected: 5000.50,
		},
		{
			name:     "String vazia",
			input:    "",
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParsePreco(tt.input)
			if got != tt.expected {
				t.Errorf("ParsePreco(%q) = %f; esperado %f", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGetRequestDelay(t *testing.T) {
	os.Unsetenv("REQUEST_DELAY_MS")
	if delay := GetRequestDelay(); delay != 1200*time.Millisecond {
		t.Errorf("delay padrão esperado 1200ms, obtido %v", delay)
	}

	os.Setenv("REQUEST_DELAY_MS", "500")
	if delay := GetRequestDelay(); delay != 500*time.Millisecond {
		t.Errorf("delay customizado esperado 500ms, obtido %v", delay)
	}
	os.Unsetenv("REQUEST_DELAY_MS")
}
