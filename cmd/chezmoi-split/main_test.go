package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
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

// Integration tests

func TestIntegration_JSON(t *testing.T) {
	script := `#!/usr/bin/env chezmoi-split
# version 1
# format json
# ignore ["app", "setting"]
#---
{
  "managed": "value",
  "app": {
    "setting": "default"
  }
}
`
	current := `{
  "managed": "old",
  "app": {
    "setting": "user-modified"
  }
}
`
	want := `{
  "managed": "value",
  "app": {
    "setting": "user-modified"
  }
}
`
	runIntegrationTest(t, script, current, want)
}

func TestIntegration_JSON_Wildcard(t *testing.T) {
	script := `#!/usr/bin/env chezmoi-split
# version 1
# format json
# ignore ["servers", "*", "enabled"]
#---
{
  "servers": {
    "server1": {"host": "managed1", "enabled": false},
    "server2": {"host": "managed2", "enabled": false}
  }
}
`
	current := `{
  "servers": {
    "server1": {"host": "old1", "enabled": true},
    "server2": {"host": "old2", "enabled": true}
  }
}
`
	// enabled should be preserved from current (true), hosts from managed
	result := runIntegrationTestGetResult(t, script, current)

	if !strings.Contains(result, `"host": "managed1"`) {
		t.Errorf("Expected managed host, got: %s", result)
	}
	if !strings.Contains(result, `"enabled": true`) {
		t.Errorf("Expected preserved enabled=true, got: %s", result)
	}
}

func TestIntegration_TOML(t *testing.T) {
	script := `#!/usr/bin/env chezmoi-split
# version 1
# format toml
# ignore ["user", "preference"]
#---
[server]
host = "localhost"
port = 8080

[user]
name = "default"
preference = "light"
`
	current := `[server]
host = "oldhost"
port = 9090

[user]
name = "oldname"
preference = "dark"
`
	result := runIntegrationTestGetResult(t, script, current)

	// Server values should come from managed
	if !strings.Contains(result, "localhost") {
		t.Errorf("Expected managed host 'localhost', got: %s", result)
	}
	if !strings.Contains(result, "8080") {
		t.Errorf("Expected managed port 8080, got: %s", result)
	}
	// User preference should be preserved from current
	if !strings.Contains(result, "dark") {
		t.Errorf("Expected preserved preference 'dark', got: %s", result)
	}
}

func TestIntegration_INI(t *testing.T) {
	script := `#!/usr/bin/env chezmoi-split
# version 1
# format ini
# ignore ["database", "password"]
#---
[database]
host = localhost
port = 3306
password = default

[server]
address = 0.0.0.0
`
	current := `[database]
host = oldhost
port = 5432
password = secret123

[server]
address = 127.0.0.1
`
	result := runIntegrationTestGetResult(t, script, current)

	// Database host/port should come from managed
	if !strings.Contains(result, "localhost") {
		t.Errorf("Expected managed host 'localhost', got: %s", result)
	}
	if !strings.Contains(result, "3306") {
		t.Errorf("Expected managed port 3306, got: %s", result)
	}
	// Password should be preserved from current
	if !strings.Contains(result, "secret123") {
		t.Errorf("Expected preserved password 'secret123', got: %s", result)
	}
}

func TestIntegration_TOML_StripCommentsError(t *testing.T) {
	script := `#!/usr/bin/env chezmoi-split
# version 1
# format toml
# strip-comments true
#---
key = "value"
`
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "script.toml")
	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("Failed to write script: %v", err)
	}

	// Redirect stdin
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	w.WriteString("")
	w.Close()
	defer func() { os.Stdin = oldStdin }()

	err := runAsInterpreter(scriptPath)
	if err == nil {
		t.Error("Expected error for strip-comments with TOML")
	}
	if !strings.Contains(err.Error(), "strip-comments") {
		t.Errorf("Expected strip-comments error, got: %v", err)
	}
}

func TestIntegration_INI_StripCommentsError(t *testing.T) {
	script := `#!/usr/bin/env chezmoi-split
# version 1
# format ini
# strip-comments true
#---
[section]
key = value
`
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "script.ini")
	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("Failed to write script: %v", err)
	}

	// Redirect stdin
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	w.WriteString("")
	w.Close()
	defer func() { os.Stdin = oldStdin }()

	err := runAsInterpreter(scriptPath)
	if err == nil {
		t.Error("Expected error for strip-comments with INI")
	}
	if !strings.Contains(err.Error(), "strip-comments") {
		t.Errorf("Expected strip-comments error, got: %v", err)
	}
}

// Helper functions

func runIntegrationTest(t *testing.T, script, current, want string) {
	t.Helper()
	result := runIntegrationTestGetResult(t, script, current)

	// Normalize whitespace for comparison
	wantNorm := strings.TrimSpace(want)
	resultNorm := strings.TrimSpace(result)

	if resultNorm != wantNorm {
		t.Errorf("Result mismatch:\ngot:\n%s\n\nwant:\n%s", resultNorm, wantNorm)
	}
}

func runIntegrationTestGetResult(t *testing.T, script, current string) string {
	t.Helper()

	// Create temp script file
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "script")
	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("Failed to write script: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Redirect stdin
	oldStdin := os.Stdin
	stdinR, stdinW, _ := os.Pipe()
	os.Stdin = stdinR
	go func() {
		stdinW.WriteString(current)
		stdinW.Close()
	}()

	// Run
	err := runAsInterpreter(scriptPath)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout
	os.Stdin = oldStdin

	if err != nil {
		t.Fatalf("runAsInterpreter failed: %v", err)
	}

	// Read captured output
	out, _ := io.ReadAll(r)
	return string(out)
}
