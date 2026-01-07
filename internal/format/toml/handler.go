// Package toml provides a TOML format handler for chezmoi-split.
package toml

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/iancoleman/orderedmap"
	"github.com/thirteen37/chezmoi-split/internal/format"
	"github.com/thirteen37/chezmoi-split/internal/path"
)

// Handler implements format.Handler for TOML files.
type Handler struct{}

// New creates a new TOML handler.
func New() *Handler {
	return &Handler{}
}

// Parse reads TOML bytes and returns an *orderedmap.OrderedMap.
// Key order from the original TOML document is preserved.
func (h *Handler) Parse(data []byte, opts format.ParseOptions) (any, error) {
	if opts.StripComments {
		return nil, fmt.Errorf("strip-comments is not supported for TOML format")
	}

	// Decode into a generic map to get values
	var raw map[string]any
	meta, err := toml.Decode(string(data), &raw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}

	// Convert to ordered map using metadata for key order
	return convertToOrderedMapWithMeta(raw, meta, nil), nil
}

// convertToOrderedMapWithMeta recursively converts map[string]any to *orderedmap.OrderedMap
// using TOML metadata to preserve key order.
func convertToOrderedMapWithMeta(v any, meta toml.MetaData, prefix []string) any {
	switch val := v.(type) {
	case map[string]any:
		result := orderedmap.New()

		// Get keys in document order from metadata
		keys := getKeysInOrder(meta, prefix, val)

		for _, k := range keys {
			childVal := val[k]
			childPrefix := append(prefix, k)
			result.Set(k, convertToOrderedMapWithMeta(childVal, meta, childPrefix))
		}
		return result
	case []map[string]any:
		// Array of tables
		result := make([]any, len(val))
		for i, item := range val {
			// For array items, we use index in prefix for nested lookups
			result[i] = convertToOrderedMapWithMeta(item, meta, prefix)
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, item := range val {
			result[i] = convertToOrderedMapWithMeta(item, meta, prefix)
		}
		return result
	default:
		return val
	}
}

// getKeysInOrder returns map keys in document order using TOML metadata.
func getKeysInOrder(meta toml.MetaData, prefix []string, m map[string]any) []string {
	// Build a set of keys we need to find
	needed := make(map[string]bool)
	for k := range m {
		needed[k] = true
	}

	// Get keys in order from metadata
	var ordered []string
	for _, key := range meta.Keys() {
		// Check if this key matches our prefix + one more segment
		if len(key) == len(prefix)+1 && matchesPrefix(key, prefix) {
			k := key[len(prefix)]
			if needed[k] && !contains(ordered, k) {
				ordered = append(ordered, k)
			}
		}
	}

	// Add any keys not found in metadata (shouldn't happen, but be safe)
	for k := range needed {
		if !contains(ordered, k) {
			ordered = append(ordered, k)
		}
	}

	return ordered
}

// matchesPrefix checks if key starts with prefix.
func matchesPrefix(key toml.Key, prefix []string) bool {
	if len(key) < len(prefix) {
		return false
	}
	for i, p := range prefix {
		if key[i] != p {
			return false
		}
	}
	return true
}

// contains checks if slice contains string.
func contains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// Serialize writes the tree to formatted TOML bytes.
func (h *Handler) Serialize(tree any, opts format.SerializeOptions) ([]byte, error) {
	// Convert ordered map to regular map for TOML encoding
	regular := convertToRegularMap(tree)

	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(regular); err != nil {
		return nil, fmt.Errorf("failed to serialize TOML: %w", err)
	}

	return buf.Bytes(), nil
}

// convertToRegularMap recursively converts *orderedmap.OrderedMap to map[string]any.
// Note: This loses key order, but BurntSushi/toml encoder sorts keys alphabetically anyway.
func convertToRegularMap(v any) any {
	switch val := v.(type) {
	case *orderedmap.OrderedMap:
		result := make(map[string]any)
		for _, k := range val.Keys() {
			v, _ := val.Get(k)
			result[k] = convertToRegularMap(v)
		}
		return result
	case orderedmap.OrderedMap:
		result := make(map[string]any)
		for _, k := range val.Keys() {
			v, _ := val.Get(k)
			result[k] = convertToRegularMap(v)
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, v := range val {
			result[i] = convertToRegularMap(v)
		}
		return result
	default:
		return val
	}
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

// FormatError returns a detailed error message for TOML parse errors.
func FormatError(content string, err error) error {
	// BurntSushi/toml errors include line numbers in the message
	// Extract and format them nicely
	errStr := err.Error()

	// Try to find line number in error message (format: "line X:")
	if strings.Contains(errStr, "line ") {
		return fmt.Errorf("TOML parse error: %w", err)
	}

	return fmt.Errorf("failed to parse TOML: %w", err)
}

// Ensure Handler implements format.Handler.
var _ format.Handler = (*Handler)(nil)
