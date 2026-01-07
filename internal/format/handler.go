// Package format provides interfaces and implementations for handling different configuration file formats.
package format

import "github.com/thirteen37/chezmoi-split/internal/path"

// ParseOptions configures parsing behavior.
type ParseOptions struct {
	StripComments bool // Strip comments (for JSON/JSONC)
}

// SerializeOptions configures serialization behavior.
type SerializeOptions struct {
	Indent string // Indentation string (e.g., "  " or "\t")
}

// Handler defines the interface for configuration file format handlers.
type Handler interface {
	// Parse reads raw bytes and returns a generic tree structure.
	Parse(data []byte, opts ParseOptions) (any, error)

	// Serialize writes the tree back to bytes.
	Serialize(tree any, opts SerializeOptions) ([]byte, error)

	// GetPath extracts a value at the given path.
	GetPath(tree any, p path.Path) (any, bool)

	// SetPath sets a value at the given path.
	SetPath(tree any, p path.Path, value any) error
}
