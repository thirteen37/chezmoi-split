package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thirteen37/chezmoi-split/internal/config"
	"github.com/thirteen37/chezmoi-split/internal/path"
)

var removePathCmd = &cobra.Command{
	Use:   "remove-path <target> <path>",
	Short: "Remove an app-owned path from a split configuration",
	Long: `Remove an app-owned path from an existing split configuration.

Arguments:
  target  Target file path relative to home (e.g., .config/zed/settings.json)
  path    JSON path array (e.g., '["agent","default_model"]')

Example:
  chezmoi split remove-path .config/zed/settings.json '["agent","default_model"]'`,
	Args: cobra.ExactArgs(2),
	RunE: runRemovePath,
}

func runRemovePath(cmd *cobra.Command, args []string) error {
	target := args[0]
	pathStr := args[1]

	// Parse path
	arrayPath, err := path.ParseArrayPath(pathStr)
	if err != nil {
		return fmt.Errorf("invalid path %q: %w", pathStr, err)
	}

	// Find paths config file
	configPath, err := findPathsConfig(target)
	if err != nil {
		return err
	}

	// Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Remove path
	if !cfg.RemovePath(arrayPath.Segments()) {
		fmt.Printf("Path %s not found\n", pathStr)
		return nil
	}

	// Save config
	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Removed path %s\n", pathStr)
	return nil
}
