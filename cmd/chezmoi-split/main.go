// chezmoi-split is a chezmoi plugin for managing co-managed configuration files.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/thirteen37/chezmoi-split/internal/format"
	"github.com/thirteen37/chezmoi-split/internal/format/json"
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

	// Read current file from stdin
	currentData, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}

	// Create handler (currently only JSON supported)
	handler := json.New()
	parseOpts := format.ParseOptions{StripComments: scr.StripComments}

	// Parse managed config from template
	managed, err := handler.Parse([]byte(scr.Template), parseOpts)
	if err != nil {
		return fmt.Errorf("failed to parse managed config: %w", err)
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
