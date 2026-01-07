// Package cmd provides the CLI commands for chezmoi-split.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "chezmoi-split",
	Short: "Manage co-managed configuration files with chezmoi",
	Long: `chezmoi-split is a chezmoi plugin that helps manage configuration files
that are co-managed by both chezmoi and an application.

It generates modify scripts that merge chezmoi-managed configuration with
application-managed paths, preserving runtime changes made by the application.`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(mergeCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(addPathCmd)
	rootCmd.AddCommand(removePathCmd)
	rootCmd.AddCommand(listCmd)
}
