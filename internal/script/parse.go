// Package script provides parsing for chezmoi-split script files.
package script

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/thirteen37/chezmoi-split/internal/path"
)

// CurrentVersion is the latest supported script format version.
const CurrentVersion = 1

// SupportedFormats lists the config formats that are currently supported.
var SupportedFormats = []string{"json", "toml", "ini", "plaintext", "auto"}

// Script represents a parsed chezmoi-split script.
type Script struct {
	Version       int
	Format        string
	StripComments bool
	IgnorePaths   []path.Path
	Header        string   // Lines before the config content (comments, etc.)
	Template      string   // The actual config content (JSON/YAML)
	Warnings      []string // Non-fatal warnings encountered during parsing
}

// Parse parses a chezmoi-split script from its content.
// Directives are prefixed with '# ' and the template section starts after '#---'.
// Lines before the actual config content (JSON/YAML) are preserved as Header.
func Parse(content string) (*Script, error) {
	script := &Script{
		Format: "auto", // default to auto-detection
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0
	versionSeen := false
	var templateLines []string
	inTemplate := false

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip shebang
		if lineNum == 1 && strings.HasPrefix(line, "#!") {
			continue
		}

		// If we're in template mode, collect all remaining lines
		if inTemplate {
			templateLines = append(templateLines, line)
			continue
		}

		trimmed := strings.TrimSpace(line)

		// Skip blank lines in directive section
		if trimmed == "" {
			continue
		}

		// Skip comment-only lines (just "#" with optional whitespace)
		if trimmed == "#" {
			continue
		}

		// Check for separator marking start of template
		if trimmed == "#---" {
			inTemplate = true
			continue
		}

		// Must be a directive line starting with "# "
		if !strings.HasPrefix(trimmed, "# ") {
			return nil, fmt.Errorf("line %d: expected directive (starting with '# ') or separator '#---', got %q", lineNum, trimmed)
		}

		// Parse directive
		directiveLine := strings.TrimPrefix(trimmed, "# ")
		parts := strings.SplitN(directiveLine, " ", 2)
		if len(parts) < 2 {
			return nil, fmt.Errorf("line %d: invalid directive %q (missing value)", lineNum, trimmed)
		}

		directive := parts[0]
		value := strings.TrimSpace(parts[1])

		switch directive {
		case "version":
			if versionSeen {
				return nil, fmt.Errorf("line %d: duplicate version directive", lineNum)
			}
			var v int
			if _, err := fmt.Sscanf(value, "%d", &v); err != nil {
				return nil, fmt.Errorf("line %d: invalid version %q", lineNum, value)
			}
			if v > CurrentVersion {
				return nil, fmt.Errorf("line %d: unsupported version %d (max supported: %d), please upgrade chezmoi-split", lineNum, v, CurrentVersion)
			}
			if v < 1 {
				return nil, fmt.Errorf("line %d: invalid version %d", lineNum, v)
			}
			script.Version = v
			versionSeen = true

		case "format":
			if !versionSeen {
				return nil, fmt.Errorf("line %d: version directive must come first", lineNum)
			}
			if !isFormatSupported(value) {
				return nil, fmt.Errorf("line %d: unsupported format %q (supported: %v)", lineNum, value, SupportedFormats)
			}
			script.Format = value

		case "strip-comments":
			if !versionSeen {
				return nil, fmt.Errorf("line %d: version directive must come first", lineNum)
			}
			switch value {
			case "true":
				script.StripComments = true
			case "false":
				script.StripComments = false
			default:
				return nil, fmt.Errorf("line %d: strip-comments must be true or false", lineNum)
			}

		case "ignore":
			if !versionSeen {
				return nil, fmt.Errorf("line %d: version directive must come first", lineNum)
			}
			p, err := path.ParseArrayPath(value)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid ignore path %q: %w", lineNum, value, err)
			}
			script.IgnorePaths = append(script.IgnorePaths, p)

		default:
			return nil, fmt.Errorf("line %d: unknown directive %q", lineNum, directive)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading script: %w", err)
	}

	if !versionSeen {
		return nil, fmt.Errorf("missing required version directive")
	}

	if len(templateLines) == 0 {
		return nil, fmt.Errorf("no template content found")
	}

	// For plaintext format, treat everything after #--- as template content
	// (no header/content separation based on config patterns)
	if script.Format == "plaintext" {
		script.Template = strings.Join(templateLines, "\n")
		// Warn about directives that don't apply to plaintext
		if len(script.IgnorePaths) > 0 {
			script.Warnings = append(script.Warnings,
				"ignore directives are not used with plaintext format; use chezmoi:ignored blocks instead")
		}
		if script.StripComments {
			script.Warnings = append(script.Warnings,
				"strip-comments is not supported for plaintext format")
		}
		return script, nil
	}

	// Separate header lines from actual config content
	header, template := splitHeaderAndContent(templateLines)
	script.Header = header
	script.Template = template

	if script.Template == "" {
		return nil, fmt.Errorf("no config content found (only header lines)")
	}

	return script, nil
}

// splitHeaderAndContent separates header lines (comments, blank lines before config)
// from the actual config content (JSON/YAML).
func splitHeaderAndContent(lines []string) (header, content string) {
	contentStart := -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isConfigStart(trimmed) {
			contentStart = i
			break
		}
	}

	if contentStart == -1 {
		// No config content found, treat everything as header
		return strings.Join(lines, "\n"), ""
	}

	if contentStart == 0 {
		// No header, all content
		return "", strings.Join(lines, "\n")
	}

	headerLines := lines[:contentStart]
	contentLines := lines[contentStart:]

	return strings.Join(headerLines, "\n"), strings.Join(contentLines, "\n")
}

// isConfigStart checks if a line looks like the start of config content.
// Detects JSON ({ or [), TOML (key = value or [section]), and INI ([section] or key = value).
func isConfigStart(line string) bool {
	// JSON object or array
	if strings.HasPrefix(line, "{") || strings.HasPrefix(line, "[") {
		return true
	}
	// TOML/INI key = value pattern (but not a comment)
	if strings.Contains(line, "=") && !strings.HasPrefix(line, "#") {
		return true
	}
	return false
}

// isFormatSupported checks if the given format is in the supported list.
func isFormatSupported(format string) bool {
	for _, f := range SupportedFormats {
		if f == format {
			return true
		}
	}
	return false
}
