// chezmoi-split is a chezmoi plugin for managing co-managed configuration files.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/thirteen37/chezmoi-split/internal/cmd"
	"github.com/thirteen37/chezmoi-split/internal/format"
	"github.com/thirteen37/chezmoi-split/internal/format/json"
	"github.com/thirteen37/chezmoi-split/internal/merge"
	"github.com/thirteen37/chezmoi-split/internal/script"
)

func main() {
	// Check if running as interpreter (shebang invocation)
	// When invoked via shebang: argv[0] = interpreter path, argv[1] = script path
	if len(os.Args) == 2 && isScriptPath(os.Args[1]) {
		if err := runAsInterpreter(os.Args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "chezmoi-split: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Normal CLI mode
	cmd.Execute()
}

// isScriptPath checks if the argument looks like a script path (not a subcommand).
func isScriptPath(arg string) bool {
	// Subcommands don't contain path separators or start with -
	if arg == "" || arg[0] == '-' {
		return false
	}
	// If it contains a path separator, it's a path
	if filepath.IsAbs(arg) || len(arg) > 0 && (arg[0] == '.' || arg[0] == '/') {
		return true
	}
	// Check if it's an existing file
	info, err := os.Stat(arg)
	if err == nil && !info.IsDir() {
		return true
	}
	return false
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

	_, err = os.Stdout.Write(output)
	return err
}
