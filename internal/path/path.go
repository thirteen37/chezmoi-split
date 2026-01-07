// Package path provides path selector abstractions for navigating configuration trees.
package path

import (
	"encoding/json"
	"fmt"
)

// Path represents a selector for navigating a configuration tree.
type Path interface {
	// Segments returns the path as a slice of string keys.
	Segments() []string

	// String returns a canonical string representation.
	String() string
}

// ArrayPath is a path specified as an array of string keys.
// Example: ["agent", "default_model"]
type ArrayPath struct {
	segments []string
}

// NewArrayPath creates a new ArrayPath from string segments.
func NewArrayPath(segments []string) *ArrayPath {
	return &ArrayPath{segments: segments}
}

// ParseArrayPath parses a JSON array string into an ArrayPath.
// Example input: `["agent", "default_model"]`
func ParseArrayPath(s string) (*ArrayPath, error) {
	var segments []string
	if err := json.Unmarshal([]byte(s), &segments); err != nil {
		return nil, fmt.Errorf("invalid path array: %w", err)
	}
	return &ArrayPath{segments: segments}, nil
}

// Segments returns the path segments.
func (p *ArrayPath) Segments() []string {
	return p.segments
}

// String returns the path as a JSON array string.
func (p *ArrayPath) String() string {
	data, _ := json.Marshal(p.segments)
	return string(data)
}
