package main

import (
	"strings"
	"testing"
)

func TestCalculateRows(t *testing.T) {
	termWidth := 80

	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "ASCII standard - 1 riga",
			input:    "0> " + strings.Repeat("a", 70), // 73 colonne visive
			expected: 1,
		},
		{
			name:     "ASCII al limite del wrapping - 2 righe",
			input:    "0> " + strings.Repeat("a", 78), // 81 colonne visive
			expected: 2,
		},
		{
			name:     "Testo accentato (40 caratteri 'é' = 40 colonne visive, ma 80 byte)",
			input:    "0> " + strings.Repeat("é", 40), // 3 + 40 = 43 colonne visive -> 1 riga
			expected: 1,
		},
		{
			name:     "Testo accentato oltre il limite",
			input:    "0> " + strings.Repeat("é", 80), // 3 + 80 = 83 colonne visive -> 2 righe
			expected: 2,
		},
		{
			name:     "Caratteri CJK (occupano 2 colonne visive ciascuno)",
			input:    "0> " + strings.Repeat("漢", 40), // 3 + (40 * 2) = 83 colonne visive -> 2 righe
			expected: 2,
		},
		{
			name:     "Emoji (occupano 2 colonne visive ciascuna)",
			input:    "0> " + strings.Repeat("😀", 40), // 3 + (40 * 2) = 83 colonne visive -> 2 righe
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateRows(tt.input, termWidth)
			if got != tt.expected {
				t.Errorf("%s: calcolato = %d, atteso = %d", tt.name, got, tt.expected)
			}
		})
	}
}
