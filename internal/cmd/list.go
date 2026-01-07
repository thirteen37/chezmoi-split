package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thirteen37/chezmoi-split/internal/config"
)

var listCmd = &cobra.Command{
	Use:   "list <target>",
	Short: "List app-owned paths for a target file",
	Long: `List all app-owned paths configured for a target file.

Arguments:
  target  Target file path relative to home (e.g., .config/zed/settings.json)

Example:
  chezmoi split list .config/zed/settings.json`,
	Args: cobra.ExactArgs(1),
	RunE: runList,
}

func runList(cmd *cobra.Command, args []string) error {
	target := args[0]

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

	// Print paths
	if len(cfg.Paths) == 0 {
		fmt.Println("No app-owned paths configured")
		return nil
	}

	fmt.Printf("App-owned paths for %s:\n", target)
	for _, p := range cfg.Paths {
		pathJSON, _ := json.Marshal(p)
		fmt.Printf("  %s\n", pathJSON)
	}

	return nil
}
