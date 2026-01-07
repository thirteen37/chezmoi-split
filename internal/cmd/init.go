package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thirteen37/chezmoi-split/internal/path"
	"github.com/thirteen37/chezmoi-split/internal/script"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a split configuration for a target file",
	Long: `Initialize a new split configuration for a target file.

This generates a single modify script file that contains:
- chezmoi-split directives (version, format, ignore paths)
- The managed configuration template (inlined)

The file uses chezmoi:modify-template to enable chezmoi templating
while being interpreted by chezmoi-split at runtime.

Example:
  chezmoi split init \
    --template zed-settings.json.tmpl \
    --target .config/zed/settings.json \
    --paths '["agent","default_model"]'

  # Or with an existing config file as template:
  chezmoi split init \
    --from ~/.config/zed/settings.json \
    --target .config/zed/settings.json \
    --paths '["agent","default_model"]'`,
	RunE: runInit,
}

var (
	templateName      string
	fromFile          string
	targetPath        string
	initialPaths      []string
	initStripComments bool
	initFormat        string
)

func init() {
	initCmd.Flags().StringVarP(&templateName, "template", "t", "", "Template name in .chezmoitemplates to inline")
	initCmd.Flags().StringVar(&fromFile, "from", "", "Existing config file to use as template")
	initCmd.Flags().StringVar(&targetPath, "target", "", "Target file path relative to home (required)")
	initCmd.Flags().StringArrayVar(&initialPaths, "paths", nil, "App-owned paths as JSON arrays (can specify multiple)")
	initCmd.Flags().BoolVar(&initStripComments, "strip-comments", false, "Enable JSON comment stripping")
	initCmd.Flags().StringVar(&initFormat, "format", "json", "Config format (json, yaml, etc.)")

	initCmd.MarkFlagRequired("target")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Validate flags
	if templateName == "" && fromFile == "" {
		return fmt.Errorf("either --template or --from must be specified")
	}
	if templateName != "" && fromFile != "" {
		return fmt.Errorf("--template and --from are mutually exclusive")
	}

	// Get chezmoi source directory
	sourceDir, err := getChezmoiSourceDir()
	if err != nil {
		return fmt.Errorf("failed to get chezmoi source dir: %w", err)
	}

	// Parse ignore paths
	var ignorePaths [][]string
	for _, p := range initialPaths {
		arrayPath, err := path.ParseArrayPath(p)
		if err != nil {
			return fmt.Errorf("invalid path %q: %w", p, err)
		}
		ignorePaths = append(ignorePaths, arrayPath.Segments())
	}

	// Get template content
	var templateContent string
	if fromFile != "" {
		// Read from existing file
		expandedPath := expandPath(fromFile)
		data, err := os.ReadFile(expandedPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", fromFile, err)
		}
		templateContent = string(data)
	} else {
		// Use template directive - will be rendered by chezmoi
		templateContent = fmt.Sprintf(`{{ template "%s" . }}`, templateName)
	}

	// Determine output file location
	targetDir := filepath.Dir(targetPath)
	targetBase := filepath.Base(targetPath)
	sourceDirPath := convertToChezmoiPath(targetDir)

	modifyScriptName := fmt.Sprintf("modify_%s.tmpl", targetBase)
	modifyScriptPath := filepath.Join(sourceDir, sourceDirPath, modifyScriptName)

	// Create directory if needed
	if err := os.MkdirAll(filepath.Join(sourceDir, sourceDirPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Generate the script content
	var sb strings.Builder
	sb.WriteString("#!/usr/bin/env chezmoi-split\n")
	sb.WriteString(fmt.Sprintf("version %d\n", script.CurrentVersion))
	sb.WriteString("\n")

	if initFormat != "" && initFormat != "auto" {
		sb.WriteString(fmt.Sprintf("format %s\n", initFormat))
	}
	if initStripComments {
		sb.WriteString("strip-comments true\n")
	}

	if len(ignorePaths) > 0 {
		sb.WriteString("\n")
		for _, p := range ignorePaths {
			pathJSON, _ := json.Marshal(p)
			sb.WriteString(fmt.Sprintf("ignore %s\n", pathJSON))
		}
	}

	sb.WriteString("\nchezmoi:modify-template\n")
	sb.WriteString(templateContent)
	if !strings.HasSuffix(templateContent, "\n") {
		sb.WriteString("\n")
	}

	// Write the file
	if err := os.WriteFile(modifyScriptPath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("failed to write modify script: %w", err)
	}

	fmt.Printf("Created: %s\n", modifyScriptPath)
	return nil
}

// getChezmoiSourceDir returns the chezmoi source directory.
func getChezmoiSourceDir() (string, error) {
	cmd := exec.Command("chezmoi", "source-path")
	output, err := cmd.Output()
	if err != nil {
		// Fallback to default
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share", "chezmoi"), nil
	}
	return strings.TrimSpace(string(output)), nil
}

// convertToChezmoiPath converts a target path to chezmoi source path.
// Example: .config/zed -> dot_config/zed
func convertToChezmoiPath(p string) string {
	parts := strings.Split(p, string(filepath.Separator))
	for i, part := range parts {
		if strings.HasPrefix(part, ".") {
			parts[i] = "dot_" + strings.TrimPrefix(part, ".")
		}
	}
	return filepath.Join(parts...)
}

// expandPath expands ~ to home directory.
func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return p
		}
		return filepath.Join(home, p[2:])
	}
	return p
}
