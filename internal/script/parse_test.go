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
		wantErr     bool
	}{
		{
			name: "basic script",
			content: `#!/usr/bin/env chezmoi-split
version 1

format json

{"key": "value"}
`,
			wantVersion: 1,
			wantFormat:  "json",
			wantPaths:   0,
		},
		{
			name: "with ignore paths",
			content: `#!/usr/bin/env chezmoi-split
version 1

format json
strip-comments true

ignore ["agent", "default_model"]
ignore ["features", "enabled"]

{"key": "value"}
`,
			wantVersion: 1,
			wantFormat:  "json",
			wantStrip:   true,
			wantPaths:   2,
		},
		{
			name: "missing version",
			content: `#!/usr/bin/env chezmoi-split

format json

{"key": "value"}
`,
			wantErr: true,
		},
		{
			name: "version not first",
			content: `#!/usr/bin/env chezmoi-split

format json
version 1

{"key": "value"}
`,
			wantErr: true,
		},
		{
			name: "unsupported version",
			content: `#!/usr/bin/env chezmoi-split
version 999

{"key": "value"}
`,
			wantErr: true,
		},
		{
			name: "no template",
			content: `#!/usr/bin/env chezmoi-split
version 1

format json
`,
			wantErr: true,
		},
		{
			name: "invalid ignore path",
			content: `#!/usr/bin/env chezmoi-split
version 1

ignore not-a-json-array

{"key": "value"}
`,
			wantErr: true,
		},
		{
			name: "auto format default",
			content: `#!/usr/bin/env chezmoi-split
version 1

{"key": "value"}
`,
			wantVersion: 1,
			wantFormat:  "auto",
			wantPaths:   0,
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
		})
	}
}

func TestParse_TemplateContent(t *testing.T) {
	content := `#!/usr/bin/env chezmoi-split
version 1

format json

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

	expected := `{
  "key": "value",
  "nested": {
    "inner": true
  }
}`
	if script.Template != expected {
		t.Errorf("Template = %q, want %q", script.Template, expected)
	}
}
