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

// Script represents a parsed chezmoi-split script.
type Script struct {
	Version       int
	Format        string
	StripComments bool
	IgnorePaths   []path.Path
	Template      string
}

// Parse parses a chezmoi-split script from its content.
// The content is expected to have the chezmoi:modify-template line already removed by chezmoi.
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

		// Skip blank lines in directive section
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Check if this line starts the template (JSON or YAML)
		if isTemplateStart(trimmed) {
			inTemplate = true
			templateLines = append(templateLines, line)
			continue
		}

		// Parse directive
		parts := strings.SplitN(trimmed, " ", 2)
		if len(parts) < 2 {
			return nil, fmt.Errorf("line %d: invalid directive %q", lineNum, trimmed)
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

	script.Template = strings.Join(templateLines, "\n")
	return script, nil
}

// isTemplateStart checks if a line looks like the start of template content.
func isTemplateStart(line string) bool {
	// JSON
	if strings.HasPrefix(line, "{") || strings.HasPrefix(line, "[") {
		return true
	}
	// YAML (common indicators)
	if strings.HasPrefix(line, "---") {
		return true
	}
	// YAML key-value (but not our directives)
	if strings.Contains(line, ":") && !isKnownDirective(strings.SplitN(line, " ", 2)[0]) {
		return true
	}
	return false
}

// isKnownDirective checks if a word is a known directive.
func isKnownDirective(word string) bool {
	switch word {
	case "version", "format", "strip-comments", "ignore":
		return true
	}
	return false
}
