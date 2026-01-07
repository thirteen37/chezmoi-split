package format

import "github.com/iancoleman/orderedmap"

// ToOrderedMapPtr converts both value and pointer types of OrderedMap to a pointer.
// Returns nil if the value is not an OrderedMap.
func ToOrderedMapPtr(v any) *orderedmap.OrderedMap {
	switch val := v.(type) {
	case *orderedmap.OrderedMap:
		return val
	case orderedmap.OrderedMap:
		return &val
	default:
		return nil
	}
}
