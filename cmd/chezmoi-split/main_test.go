package main

import (
	"testing"
)

func TestGetErrorContext(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		offset     int
		wantLine   int
		wantCol    int
		wantInSnip string
	}{
		{
			name:       "first line error",
			content:    `{"key": value}`,
			offset:     9,
			wantLine:   1,
			wantCol:    10,
			wantInSnip: "value",
		},
		{
			name:       "second line error",
			content:    "{\n  \"key\": value\n}",
			offset:     12,
			wantLine:   2,
			wantCol:    11,
			wantInSnip: "value",
		},
		{
			name:       "offset at start",
			content:    "invalid",
			offset:     0,
			wantLine:   1,
			wantCol:    1,
			wantInSnip: "invalid",
		},
		{
			name:       "empty content",
			content:    "",
			offset:     0,
			wantLine:   1,
			wantCol:    1,
			wantInSnip: "",
		},
		{
			name:       "offset beyond content",
			content:    "short",
			offset:     100,
			wantLine:   1,
			wantCol:    1,
			wantInSnip: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line, col, snippet := getErrorContext(tt.content, tt.offset)
			if line != tt.wantLine {
				t.Errorf("line = %d, want %d", line, tt.wantLine)
			}
			if col != tt.wantCol {
				t.Errorf("col = %d, want %d", col, tt.wantCol)
			}
			if tt.wantInSnip != "" && !contains(snippet, tt.wantInSnip) {
				t.Errorf("snippet = %q, want it to contain %q", snippet, tt.wantInSnip)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && searchSubstring(s, substr)))
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
