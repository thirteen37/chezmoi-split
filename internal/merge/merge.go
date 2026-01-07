// Package merge provides the core configuration merging logic.
package merge

import (
	"reflect"

	"github.com/iancoleman/orderedmap"
	"github.com/thirteen37/chezmoi-split/internal/format"
	"github.com/thirteen37/chezmoi-split/internal/path"
)

// Merge combines a managed configuration with the current configuration,
// preserving values at app-owned paths from current.
//
// Algorithm:
// 1. Start with a deep copy of managed config
// 2. For each app-owned path:
//   - If the path exists in current, copy that value to result
//   - If the path doesn't exist in current, keep managed value
func Merge(handler format.Handler, managed, current any, paths []path.Path) any {
	// Deep copy managed to avoid modifying original
	result := deepCopy(managed)

	// If no current config, just return managed
	// Note: We check for typed nil (e.g., (*orderedmap.OrderedMap)(nil))
	// because interface comparison with nil may fail for typed nil pointers
	if isNilValue(current) {
		return result
	}

	// For each app-owned path, overlay value from current if it exists
	for _, p := range paths {
		if val, ok := handler.GetPath(current, p); ok {
			// Ignore errors - if we can't set, we skip
			_ = handler.SetPath(result, p, val)
		}
	}

	return result
}

// deepCopy creates a deep copy of a value.
// Works with ordered maps and slices typically found in JSON structures.
func deepCopy(v any) any {
	switch val := v.(type) {
	case *orderedmap.OrderedMap:
		result := orderedmap.New()
		for _, k := range val.Keys() {
			v, _ := val.Get(k)
			result.Set(k, deepCopy(v))
		}
		return result
	case orderedmap.OrderedMap:
		result := orderedmap.New()
		for _, k := range val.Keys() {
			v, _ := val.Get(k)
			result.Set(k, deepCopy(v))
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, v := range val {
			result[i] = deepCopy(v)
		}
		return result
	default:
		// Primitives (string, float64, bool, nil) are immutable
		return val
	}
}

// isNilValue checks if v is nil, including typed nil pointers inside interfaces.
func isNilValue(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Interface:
		return rv.IsNil()
	}
	return false
}
