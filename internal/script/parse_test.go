package script

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantVersion int
		wantFormat  string
		wantStrip   bool
		wantPaths   int
		wantHeader  string
		wantErr     bool
	}{
		{
			name: "basic script",
			content: `#!/usr/bin/env chezmoi-split
# version 1
# format json
#---
{"key": "value"}
`,
			wantVersion: 1,
			wantFormat:  "json",
			wantPaths:   0,
			wantHeader:  "",
		},
		{
			name: "with ignore paths",
			content: `#!/usr/bin/env chezmoi-split
# version 1
# format json
# strip-comments true
# ignore ["agent", "default_model"]
# ignore ["features", "enabled"]
#---
{"key": "value"}
`,
			wantVersion: 1,
			wantFormat:  "json",
			wantStrip:   true,
			wantPaths:   2,
			wantHeader:  "",
		},
		{
			name: "missing version",
			content: `#!/usr/bin/env chezmoi-split
# format json
#---
{"key": "value"}
`,
			wantErr: true,
		},
		{
			name: "version not first",
			content: `#!/usr/bin/env chezmoi-split
# format json
# version 1
#---
{"key": "value"}
`,
			wantErr: true,
		},
		{
			name: "unsupported version",
			content: `#!/usr/bin/env chezmoi-split
# version 999
#---
{"key": "value"}
`,
			wantErr: true,
		},
		{
			name: "no template",
			content: `#!/usr/bin/env chezmoi-split
# version 1
# format json
`,
			wantErr: true,
		},
		{
			name: "invalid ignore path",
			content: `#!/usr/bin/env chezmoi-split
# version 1
# ignore not-a-json-array
#---
{"key": "value"}
`,
			wantErr: true,
		},
		{
			name: "auto format default",
			content: `#!/usr/bin/env chezmoi-split
# version 1
#---
{"key": "value"}
`,
			wantVersion: 1,
			wantFormat:  "auto",
			wantPaths:   0,
			wantHeader:  "",
		},
		{
			name: "with header comment",
			content: `#!/usr/bin/env chezmoi-split
# version 1
# format json
#---
// This is a comment in the JSON
{"key": "value"}
`,
			wantVersion: 1,
			wantFormat:  "json",
			wantPaths:   0,
			wantHeader:  "// This is a comment in the JSON",
		},
		{
			name: "with multi-line header",
			content: `#!/usr/bin/env chezmoi-split
# version 1
# format json
#---
// First comment line
// Second comment line

{"key": "value"}
`,
			wantVersion: 1,
			wantFormat:  "json",
			wantPaths:   0,
			wantHeader:  "// First comment line\n// Second comment line\n",
		},
		{
			name: "empty comment lines in directives",
			content: `#!/usr/bin/env chezmoi-split
# version 1
#
# format json
#---
{"key": "value"}
`,
			wantVersion: 1,
			wantFormat:  "json",
			wantPaths:   0,
			wantHeader:  "",
		},
		{
			name: "missing separator",
			content: `#!/usr/bin/env chezmoi-split
# version 1
# format json
{"key": "value"}
`,
			wantErr: true,
		},
		{
			name: "invalid line without hash prefix",
			content: `#!/usr/bin/env chezmoi-split
# version 1
format json
#---
{"key": "value"}
`,
			wantErr: true,
		},
		{
			name: "unsupported format",
			content: `#!/usr/bin/env chezmoi-split
# version 1
# format yaml
#---
{"key": "value"}
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script, err := Parse(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if script.Version != tt.wantVersion {
				t.Errorf("Version = %d, want %d", script.Version, tt.wantVersion)
			}
			if script.Format != tt.wantFormat {
				t.Errorf("Format = %q, want %q", script.Format, tt.wantFormat)
			}
			if script.StripComments != tt.wantStrip {
				t.Errorf("StripComments = %v, want %v", script.StripComments, tt.wantStrip)
			}
			if len(script.IgnorePaths) != tt.wantPaths {
				t.Errorf("len(IgnorePaths) = %d, want %d", len(script.IgnorePaths), tt.wantPaths)
			}
			if script.Header != tt.wantHeader {
				t.Errorf("Header = %q, want %q", script.Header, tt.wantHeader)
			}
		})
	}
}

func TestParse_TemplateContent(t *testing.T) {
	content := `#!/usr/bin/env chezmoi-split
# version 1
# format json
#---
{
  "key": "value",
  "nested": {
    "inner": true
  }
}
`
	script, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	expectedTemplate := `{
  "key": "value",
  "nested": {
    "inner": true
  }
}`
	if script.Template != expectedTemplate {
		t.Errorf("Template = %q, want %q", script.Template, expectedTemplate)
	}
	if script.Header != "" {
		t.Errorf("Header = %q, want empty", script.Header)
	}
}

func TestParse_HeaderAndTemplate(t *testing.T) {
	content := `#!/usr/bin/env chezmoi-split
# version 1
# format json
#---
// Configuration for my application
// Do not edit the ignore paths manually
{
  "key": "value"
}
`
	script, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	expectedHeader := `// Configuration for my application
// Do not edit the ignore paths manually`
	if script.Header != expectedHeader {
		t.Errorf("Header = %q, want %q", script.Header, expectedHeader)
	}

	expectedTemplate := `{
  "key": "value"
}`
	if script.Template != expectedTemplate {
		t.Errorf("Template = %q, want %q", script.Template, expectedTemplate)
	}
}

func TestParse_PlaintextFormat(t *testing.T) {
	content := `#!/usr/bin/env chezmoi-split
# version 1
# format plaintext
#---
# chezmoi:managed
set number
set expandtab

# chezmoi:ignored
colorscheme gruvbox

# chezmoi:end
`
	script, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if script.Format != "plaintext" {
		t.Errorf("Format = %q, want %q", script.Format, "plaintext")
	}
	// For plaintext, Header should be empty (no header/content split)
	if script.Header != "" {
		t.Errorf("Header = %q, want empty for plaintext", script.Header)
	}
	// Template should contain everything after #---
	if !contains(script.Template, "chezmoi:managed") {
		t.Errorf("Template should contain chezmoi:managed marker")
	}
	if !contains(script.Template, "set number") {
		t.Errorf("Template should contain 'set number'")
	}
}

func TestParse_PlaintextWithIgnoreWarning(t *testing.T) {
	content := `#!/usr/bin/env chezmoi-split
# version 1
# format plaintext
# ignore ["should", "warn"]
#---
# chezmoi:managed
some content
# chezmoi:end
`
	script, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(script.Warnings) == 0 {
		t.Error("Expected warning for ignore directive with plaintext format")
	}

	foundWarning := false
	for _, w := range script.Warnings {
		if contains(w, "ignore") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Errorf("Expected warning about ignore directives, got: %v", script.Warnings)
	}
}

func TestParse_PlaintextWithStripCommentsWarning(t *testing.T) {
	content := `#!/usr/bin/env chezmoi-split
# version 1
# format plaintext
# strip-comments true
#---
# chezmoi:managed
some content
# chezmoi:end
`
	script, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(script.Warnings) == 0 {
		t.Error("Expected warning for strip-comments with plaintext format")
	}

	foundWarning := false
	for _, w := range script.Warnings {
		if contains(w, "strip-comments") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Errorf("Expected warning about strip-comments, got: %v", script.Warnings)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
