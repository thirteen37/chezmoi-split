// Package json provides a JSON format handler for chezmoi-split.
package json

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/iancoleman/orderedmap"
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

// Parse reads JSON bytes and returns an *orderedmap.OrderedMap.
// All nested objects are also converted to OrderedMaps to preserve key order.
func (h *Handler) Parse(data []byte, opts format.ParseOptions) (any, error) {
	if opts.StripComments {
		data = StripComments(data)
	}

	result := orderedmap.New()
	if err := json.Unmarshal(data, result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	// Convert nested map[string]interface{} to *orderedmap.OrderedMap
	return convertNestedMaps(result), nil
}

// convertNestedMaps recursively processes nested maps to ensure they're all OrderedMaps.
// The orderedmap library already handles this during unmarshal, but we process arrays too.
func convertNestedMaps(v any) any {
	switch val := v.(type) {
	case *orderedmap.OrderedMap:
		for _, k := range val.Keys() {
			v, _ := val.Get(k)
			val.Set(k, convertNestedMaps(v))
		}
		return val
	case orderedmap.OrderedMap:
		for _, k := range val.Keys() {
			v, _ := val.Get(k)
			val.Set(k, convertNestedMaps(v))
		}
		return val
	case []interface{}:
		for i, v := range val {
			val[i] = convertNestedMaps(v)
		}
		return val
	default:
		return val
	}
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
		om := toOrderedMapPtr(current)
		if om == nil {
			return nil, false
		}
		val, exists := om.Get(segment)
		if !exists {
			return nil, false
		}
		current = val
	}
	return current, true
}

// toOrderedMapPtr converts both value and pointer types of OrderedMap to a pointer.
func toOrderedMapPtr(v any) *orderedmap.OrderedMap {
	switch val := v.(type) {
	case *orderedmap.OrderedMap:
		return val
	case orderedmap.OrderedMap:
		return &val
	default:
		return nil
	}
}

// SetPath sets a value at the given path.
// Creates intermediate maps as needed.
func (h *Handler) SetPath(tree any, p path.Path, value any) error {
	segments := p.Segments()
	if len(segments) == 0 {
		return fmt.Errorf("empty path")
	}

	om := toOrderedMapPtr(tree)
	if om == nil {
		return fmt.Errorf("tree is not an ordered map")
	}

	// Navigate to parent, creating intermediate maps as needed
	for _, segment := range segments[:len(segments)-1] {
		next, exists := om.Get(segment)
		if !exists {
			next = orderedmap.New()
			om.Set(segment, next)
		}
		nextMap := toOrderedMapPtr(next)
		if nextMap == nil {
			return fmt.Errorf("path segment %q is not a map", segment)
		}
		om = nextMap
	}

	// Set the final value
	om.Set(segments[len(segments)-1], value)
	return nil
}

// Ensure Handler implements format.Handler.
var _ format.Handler = (*Handler)(nil)
