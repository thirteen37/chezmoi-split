// chezmoi-split is a chezmoi plugin for managing co-managed configuration files.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/thirteen37/chezmoi-split/internal/format"
	formatini "github.com/thirteen37/chezmoi-split/internal/format/ini"
	formatjson "github.com/thirteen37/chezmoi-split/internal/format/json"
	formatplaintext "github.com/thirteen37/chezmoi-split/internal/format/plaintext"
	formattoml "github.com/thirteen37/chezmoi-split/internal/format/toml"
	"github.com/thirteen37/chezmoi-split/internal/merge"
	"github.com/thirteen37/chezmoi-split/internal/script"
)

const usage = `chezmoi-split - merge chezmoi-managed config with app-managed paths

This tool is designed to be used as a script interpreter via shebang.
Create a modify script in your chezmoi source directory:

  ~/.local/share/chezmoi/dot_config/app/modify_settings.json.tmpl

With contents like:

  #!/usr/bin/env chezmoi-split
  # version 1
  # format json
  # ignore ["path", "to", "preserve"]
  #---
  {
    "your": "config",
    "with": "{{ .chezmoi.templates }}"
  }

See https://github.com/thirteen37/chezmoi-split for full documentation.
`

func main() {
	// Interpreter mode: argv[0] = interpreter, argv[1] = script path
	if len(os.Args) == 2 {
		if err := runAsInterpreter(os.Args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "chezmoi-split: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// No script provided - show usage
	fmt.Print(usage)
}

// runAsInterpreter executes the merge logic when invoked via shebang.
func runAsInterpreter(scriptPath string) error {
	// Read script content
	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to read script: %w", err)
	}

	// Parse script
	scr, err := script.Parse(string(scriptContent))
	if err != nil {
		return fmt.Errorf("failed to parse script: %w", err)
	}

	// Print any warnings from parsing
	for _, warning := range scr.Warnings {
		fmt.Fprintf(os.Stderr, "chezmoi-split: warning: %s\n", warning)
	}

	// Read current file from stdin
	currentData, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}

	// Handle plaintext format separately (uses block-based merging)
	if scr.Format == "plaintext" {
		return runPlaintextMerge(scr, currentData)
	}

	// Create handler based on format
	handler := getHandler(scr.Format)
	parseOpts := format.ParseOptions{StripComments: scr.StripComments}

	// Parse managed config from template
	managed, err := handler.Parse([]byte(scr.Template), parseOpts)
	if err != nil {
		return formatJSONError("managed config (in script)", scr.Template, err)
	}

	// Parse current config (may be empty)
	var current any
	if len(currentData) > 0 {
		current, err = handler.Parse(currentData, parseOpts)
		if err != nil {
			// If current is invalid, just use managed
			current = nil
		}
	}

	// Merge
	result := merge.Merge(handler, managed, current, scr.IgnorePaths)

	// Serialize and output
	output, err := handler.Serialize(result, format.SerializeOptions{})
	if err != nil {
		return fmt.Errorf("failed to serialize result: %w", err)
	}

	// Output header (comments before config) if present
	if scr.Header != "" {
		fmt.Println(scr.Header)
	}

	_, err = os.Stdout.Write(output)
	return err
}

// runPlaintextMerge handles plaintext format using block-based merging.
func runPlaintextMerge(scr *script.Script, currentData []byte) error {
	handler := formatplaintext.New()

	// Parse managed (template)
	// Note: For plaintext format, script.Template contains everything after #---
	// (the parser doesn't use header/content separation for plaintext)
	managedAny, err := handler.Parse([]byte(scr.Template), format.ParseOptions{})
	if err != nil {
		return fmt.Errorf("failed to parse managed config: %w", err)
	}
	managed := managedAny.(*formatplaintext.ParsedConfig)

	// Parse current (may be empty or have no markers)
	var current *formatplaintext.ParsedConfig
	if len(currentData) > 0 {
		currentAny, err := handler.Parse(currentData, format.ParseOptions{})
		if err == nil {
			current = currentAny.(*formatplaintext.ParsedConfig)
		}
		// Ignore parse errors - current may have no markers
	}

	// Merge using block-based logic
	result := handler.MergeBlocks(managed, current)

	// Serialize and output
	output, err := handler.Serialize(result, format.SerializeOptions{})
	if err != nil {
		return fmt.Errorf("failed to serialize: %w", err)
	}

	_, err = os.Stdout.Write(output)
	return err
}

// formatJSONError creates a more helpful error message for JSON parse errors.
func formatJSONError(context, content string, err error) error {
	// Try to extract position from JSON syntax error
	if syntaxErr, ok := err.(*json.SyntaxError); ok {
		offset := int(syntaxErr.Offset)
		line, col, snippet := getErrorContext(content, offset)
		return fmt.Errorf("failed to parse %s: %v\n  at line %d, column %d:\n  %s", context, syntaxErr, line, col, snippet)
	}

	// Generic error
	return fmt.Errorf("failed to parse %s: %w", context, err)
}

// getErrorContext returns line number, column, and a snippet around the error position.
func getErrorContext(content string, offset int) (line, col int, snippet string) {
	if offset < 0 || offset > len(content) {
		return 1, 1, ""
	}

	// Count lines and find column
	line = 1
	lineStart := 0
	for i := 0; i < offset && i < len(content); i++ {
		if content[i] == '\n' {
			line++
			lineStart = i + 1
		}
	}
	col = offset - lineStart + 1

	// Extract the line containing the error
	lineEnd := lineStart
	for lineEnd < len(content) && content[lineEnd] != '\n' {
		lineEnd++
	}

	currentLine := content[lineStart:lineEnd]

	// Create snippet with pointer
	pointer := strings.Repeat(" ", col-1) + "^"
	snippet = currentLine + "\n  " + pointer

	return line, col, snippet
}

// getHandler returns the appropriate format handler based on format name.
func getHandler(formatName string) format.Handler {
	switch formatName {
	case "toml":
		return formattoml.New()
	case "ini":
		return formatini.New()
	default:
		// "json" and "auto" both use JSON handler
		return formatjson.New()
	}
}
