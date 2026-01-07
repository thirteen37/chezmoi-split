// Package config provides configuration file handling for chezmoi-split.
package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/thirteen37/chezmoi-split/internal/path"
)

// SplitConfig represents the .split-*.json configuration file.
type SplitConfig struct {
	// Paths is a list of app-owned paths.
	// Each path is a JSON array of string keys.
	Paths [][]string `json:"paths"`

	// Options contains format-specific options.
	Options Options `json:"options,omitempty"`
}

// Options contains optional settings.
type Options struct {
	StripComments bool `json:"stripComments,omitempty"`
}

// Load reads a SplitConfig from a file.
func Load(filename string) (*SplitConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg SplitConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// Save writes the SplitConfig to a file.
func (c *SplitConfig) Save(filename string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filename, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetPaths returns the paths as path.Path objects.
func (c *SplitConfig) GetPaths() []path.Path {
	result := make([]path.Path, len(c.Paths))
	for i, p := range c.Paths {
		result[i] = path.NewArrayPath(p)
	}
	return result
}

// AddPath adds a new path to the configuration.
// Returns true if the path was added, false if it already exists.
func (c *SplitConfig) AddPath(p []string) bool {
	// Check if path already exists
	for _, existing := range c.Paths {
		if pathsEqual(existing, p) {
			return false
		}
	}
	c.Paths = append(c.Paths, p)
	return true
}

// RemovePath removes a path from the configuration.
// Returns true if the path was removed, false if it wasn't found.
func (c *SplitConfig) RemovePath(p []string) bool {
	for i, existing := range c.Paths {
		if pathsEqual(existing, p) {
			c.Paths = append(c.Paths[:i], c.Paths[i+1:]...)
			return true
		}
	}
	return false
}

// pathsEqual checks if two paths are equal.
func pathsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
