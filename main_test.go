package main

import (
	"strings"
	"testing"
)

func TestTerminalRowsUsesDisplayWidth(t *testing.T) {
	tests := []struct {
		name  string
		input string
		width int
		want  int
	}{
		{name: "empty input", width: 80, want: 1},
		{name: "ASCII at boundary", input: strings.Repeat("a", 77), width: 80, want: 1},
		{name: "ASCII past boundary", input: strings.Repeat("a", 78), width: 80, want: 2},
		{name: "accented text counts as one cell", input: strings.Repeat("é", 40), width: 80, want: 1},
		{name: "combining sequence counts as one cell", input: strings.Repeat("e\u0301", 77), width: 80, want: 1},
		{name: "CJK characters count as two cells", input: strings.Repeat("界", 39), width: 80, want: 2},
		{name: "emoji clusters count by display width", input: strings.Repeat("👩‍💻", 39), width: 80, want: 2},
		{
			name:  "wide character wrapping preserves unused cells",
			input: strings.Repeat("界", 39) + strings.Repeat("a", 79),
			width: 80,
			want:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := terminalRows(terminalPrompt, []rune(tt.input), tt.width)
			if got != tt.want {
				t.Fatalf("terminalRows() = %d, want %d", got, tt.want)
			}
		})
	}
}
