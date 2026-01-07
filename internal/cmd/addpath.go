package cmd

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thirteen37/chezmoi-split/internal/config"
	"github.com/thirteen37/chezmoi-split/internal/path"
)

var addPathCmd = &cobra.Command{
	Use:   "add-path <target> <path>",
	Short: "Add an app-owned path to a split configuration",
	Long: `Add a new app-owned path to an existing split configuration.

Arguments:
  target  Target file path relative to home (e.g., .config/zed/settings.json)
  path    JSON path array (e.g., '["agent","default_model"]')

Example:
  chezmoi split add-path .config/zed/settings.json '["context_servers","OpenDia","enabled"]'`,
	Args: cobra.ExactArgs(2),
	RunE: runAddPath,
}

func runAddPath(cmd *cobra.Command, args []string) error {
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

	// Add path
	if !cfg.AddPath(arrayPath.Segments()) {
		fmt.Printf("Path %s already exists\n", pathStr)
		return nil
	}

	// Save config
	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Added path %s\n", pathStr)
	return nil
}

// findPathsConfig finds the paths config file for a target.
func findPathsConfig(target string) (string, error) {
	sourceDir, err := getChezmoiSourceDir()
	if err != nil {
		return "", err
	}

	// Convert target path to chezmoi source path
	targetDir := filepath.Dir(target)
	targetBase := filepath.Base(target)
	sourceDirPath := convertToChezmoiPath(targetDir)

	// Paths config: .split-settings.json
	pathsConfigName := fmt.Sprintf(".split-%s", targetBase)
	configPath := filepath.Join(sourceDir, sourceDirPath, pathsConfigName)

	return configPath, nil
}

// getChezmoiSourceDir returns the chezmoi source directory.
func getChezmoiSourceDirForPath() (string, error) {
	cmd := exec.Command("chezmoi", "source-path")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get chezmoi source dir: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
