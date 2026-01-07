// Package json provides a JSON format handler for chezmoi-split.
package json

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/thirteen37/chezmoi-split/internal/format"
	"github.com/thirteen37/chezmoi-split/internal/path"
)

// Handler implements format.Handler for JSON/JSONC files.
type Handler struct{}

// New creates a new JSON handler.
func New() *Handler {
	return &Handler{}
}

// commentRegex matches single-line // comments.
var commentRegex = regexp.MustCompile(`(?m)^\s*//.*$|//[^"]*$`)

// StripComments removes single-line // comments from JSON.
// This allows parsing JSONC (JSON with comments) files.
func StripComments(data []byte) []byte {
	return commentRegex.ReplaceAll(data, nil)
}

// Parse reads JSON bytes and returns a map[string]any.
func (h *Handler) Parse(data []byte, opts format.ParseOptions) (any, error) {
	if opts.StripComments {
		data = StripComments(data)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return result, nil
}

// Serialize writes the tree to formatted JSON bytes.
func (h *Handler) Serialize(tree any, opts format.SerializeOptions) ([]byte, error) {
	indent := opts.Indent
	if indent == "" {
		indent = "  "
	}

	data, err := json.MarshalIndent(tree, "", indent)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize JSON: %w", err)
	}
	// Add trailing newline
	return append(data, '\n'), nil
}

// GetPath extracts a value at the given path.
func (h *Handler) GetPath(tree any, p path.Path) (any, bool) {
	current := tree
	for _, segment := range p.Segments() {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		val, exists := m[segment]
		if !exists {
			return nil, false
		}
		current = val
	}
	return current, true
}

// SetPath sets a value at the given path.
// Creates intermediate maps as needed.
func (h *Handler) SetPath(tree any, p path.Path, value any) error {
	segments := p.Segments()
	if len(segments) == 0 {
		return fmt.Errorf("empty path")
	}

	m, ok := tree.(map[string]any)
	if !ok {
		return fmt.Errorf("tree is not a map")
	}

	// Navigate to parent, creating intermediate maps as needed
	for _, segment := range segments[:len(segments)-1] {
		next, exists := m[segment]
		if !exists {
			next = make(map[string]any)
			m[segment] = next
		}
		nextMap, ok := next.(map[string]any)
		if !ok {
			return fmt.Errorf("path segment %q is not a map", segment)
		}
		m = nextMap
	}

	// Set the final value
	m[segments[len(segments)-1]] = value
	return nil
}

// Ensure Handler implements format.Handler.
var _ format.Handler = (*Handler)(nil)
