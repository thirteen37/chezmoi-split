package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/thirteen37/chezmoi-split/internal/config"
	"github.com/thirteen37/chezmoi-split/internal/format"
	"github.com/thirteen37/chezmoi-split/internal/format/json"
	"github.com/thirteen37/chezmoi-split/internal/merge"
)

var mergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "Merge managed and current configurations",
	Long: `Merge a managed configuration with the current configuration,
preserving app-owned paths from current.

This command is typically called by generated modify scripts.
It reads the current file from stdin and outputs the merged result to stdout.`,
	RunE: runMerge,
}

var (
	managedFile   string
	pathsFile     string
	stripComments bool
)

func init() {
	mergeCmd.Flags().StringVarP(&managedFile, "managed", "m", "", "Path to managed config file (required)")
	mergeCmd.Flags().StringVarP(&pathsFile, "paths", "p", "", "Path to paths config file (required)")
	mergeCmd.Flags().BoolVar(&stripComments, "strip-comments", false, "Strip // comments from JSON")

	mergeCmd.MarkFlagRequired("managed")
	mergeCmd.MarkFlagRequired("paths")
}

func runMerge(cmd *cobra.Command, args []string) error {
	// Read managed config
	managedData, err := os.ReadFile(managedFile)
	if err != nil {
		return fmt.Errorf("failed to read managed file: %w", err)
	}

	// Read current config from stdin
	currentData, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read current from stdin: %w", err)
	}

	// Load paths config
	cfg, err := config.Load(pathsFile)
	if err != nil {
		return fmt.Errorf("failed to load paths config: %w", err)
	}

	// Use strip-comments flag, or from config
	shouldStripComments := stripComments || cfg.Options.StripComments

	// Create handler
	handler := json.New()
	parseOpts := format.ParseOptions{StripComments: shouldStripComments}

	// Parse managed config
	managed, err := handler.Parse(managedData, parseOpts)
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
	result := merge.Merge(handler, managed, current, cfg.GetPaths())

	// Serialize and output
	output, err := handler.Serialize(result, format.SerializeOptions{})
	if err != nil {
		return fmt.Errorf("failed to serialize result: %w", err)
	}

	_, err = os.Stdout.Write(output)
	return err
}
