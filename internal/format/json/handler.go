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

// GetPath extracts a value at the given path, supporting wildcards.
func (h *Handler) GetPath(tree any, p path.Path) (any, bool) {
	return getPathWithWildcard(tree, p.Segments(), 0)
}

// getPathWithWildcard recursively navigates the tree, handling wildcards.
func getPathWithWildcard(current any, segments []string, idx int) (any, bool) {
	if idx >= len(segments) {
		return current, true
	}

	segment := segments[idx]
	om := format.ToOrderedMapPtr(current)
	if om == nil {
		return nil, false
	}

	if segment == "*" {
		// Wildcard: return first match from any key
		for _, key := range om.Keys() {
			val, _ := om.Get(key)
			if result, ok := getPathWithWildcard(val, segments, idx+1); ok {
				return result, true
			}
		}
		return nil, false
	}

	val, exists := om.Get(segment)
	if !exists {
		return nil, false
	}
	return getPathWithWildcard(val, segments, idx+1)
}

// SetPath sets a value at the given path, supporting wildcards.
// Creates intermediate maps as needed.
func (h *Handler) SetPath(tree any, p path.Path, value any) error {
	segments := p.Segments()
	if len(segments) == 0 {
		return fmt.Errorf("empty path")
	}

	return setPathWithWildcard(tree, segments, 0, value)
}

// setPathWithWildcard recursively sets values, handling wildcards.
func setPathWithWildcard(current any, segments []string, idx int, value any) error {
	if idx >= len(segments) {
		return nil
	}

	om := format.ToOrderedMapPtr(current)
	if om == nil {
		return fmt.Errorf("cannot navigate into non-map value")
	}

	segment := segments[idx]
	isLast := idx == len(segments)-1

	if segment == "*" {
		// Wildcard: apply to all keys
		for _, key := range om.Keys() {
			val, _ := om.Get(key)
			if isLast {
				om.Set(key, value)
			} else {
				if err := setPathWithWildcard(val, segments, idx+1, value); err != nil {
					// Continue to other keys even if one fails
					continue
				}
			}
		}
		return nil
	}

	if isLast {
		om.Set(segment, value)
		return nil
	}

	// Navigate deeper, creating intermediate maps if needed
	next, exists := om.Get(segment)
	if !exists {
		next = orderedmap.New()
		om.Set(segment, next)
	}

	nextMap := format.ToOrderedMapPtr(next)
	if nextMap == nil {
		return fmt.Errorf("path segment %q is not a map", segment)
	}

	return setPathWithWildcard(nextMap, segments, idx+1, value)
}

// Ensure Handler implements format.Handler.
var _ format.Handler = (*Handler)(nil)
