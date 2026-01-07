// Package ini provides an INI format handler for chezmoi-split.
package ini

import (
	"bytes"
	"fmt"

	"github.com/iancoleman/orderedmap"
	"github.com/thirteen37/chezmoi-split/internal/format"
	"github.com/thirteen37/chezmoi-split/internal/path"
	"gopkg.in/ini.v1"
)

// Handler implements format.Handler for INI files.
type Handler struct{}

// New creates a new INI handler.
func New() *Handler {
	return &Handler{}
}

// Parse reads INI bytes and returns an *orderedmap.OrderedMap.
// Structure: {"section": {"key": "value"}}
// Global keys (before any section) are stored under the empty string key "".
func (h *Handler) Parse(data []byte, opts format.ParseOptions) (any, error) {
	if opts.StripComments {
		return nil, fmt.Errorf("strip-comments is not supported for INI format")
	}

	cfg, err := ini.Load(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse INI: %w", err)
	}

	result := orderedmap.New()

	for _, section := range cfg.Sections() {
		sectionName := section.Name()
		// ini.v1 uses "DEFAULT" for global section, we use ""
		if sectionName == "DEFAULT" {
			sectionName = ""
		}

		sectionMap := orderedmap.New()
		for _, key := range section.Keys() {
			sectionMap.Set(key.Name(), key.Value())
		}

		// Only add section if it has keys (or is explicitly named)
		if len(sectionMap.Keys()) > 0 || sectionName != "" {
			result.Set(sectionName, sectionMap)
		}
	}

	return result, nil
}

// Serialize writes the tree to formatted INI bytes.
func (h *Handler) Serialize(tree any, opts format.SerializeOptions) ([]byte, error) {
	om := format.ToOrderedMapPtr(tree)
	if om == nil {
		return nil, fmt.Errorf("tree is not an ordered map")
	}

	cfg := ini.Empty()

	for _, sectionName := range om.Keys() {
		sectionVal, _ := om.Get(sectionName)
		sectionMap := format.ToOrderedMapPtr(sectionVal)
		if sectionMap == nil {
			continue
		}

		// Get or create section
		var section *ini.Section
		if sectionName == "" {
			section = cfg.Section("DEFAULT")
		} else {
			var err error
			section, err = cfg.NewSection(sectionName)
			if err != nil {
				return nil, fmt.Errorf("failed to create section %q: %w", sectionName, err)
			}
		}

		for _, keyName := range sectionMap.Keys() {
			keyVal, _ := sectionMap.Get(keyName)
			strVal := toString(keyVal)
			_, err := section.NewKey(keyName, strVal)
			if err != nil {
				return nil, fmt.Errorf("failed to create key %q: %w", keyName, err)
			}
		}
	}

	var buf bytes.Buffer
	_, err := cfg.WriteTo(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize INI: %w", err)
	}

	return buf.Bytes(), nil
}

// toString converts any value to its string representation.
// INI files only support string values.
func toString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// GetPath extracts a value at the given path, supporting wildcards.
// INI paths are limited to ["section", "key"] format (max 2 segments).
// Wildcard "*" can be used for section to match any section.
func (h *Handler) GetPath(tree any, p path.Path) (any, bool) {
	segments := p.Segments()
	if len(segments) == 0 || len(segments) > 2 {
		return nil, false
	}

	om := format.ToOrderedMapPtr(tree)
	if om == nil {
		return nil, false
	}

	sectionSegment := segments[0]

	// Handle wildcard for section
	if sectionSegment == "*" {
		// Try all sections
		for _, sectionName := range om.Keys() {
			sectionVal, _ := om.Get(sectionName)
			if len(segments) == 1 {
				return sectionVal, true
			}
			// Get key from section
			sectionMap := format.ToOrderedMapPtr(sectionVal)
			if sectionMap == nil {
				continue
			}
			keySegment := segments[1]
			if keySegment == "*" {
				// Return first key from first section
				for _, keyName := range sectionMap.Keys() {
					val, _ := sectionMap.Get(keyName)
					return val, true
				}
			} else {
				if val, exists := sectionMap.Get(keySegment); exists {
					return val, true
				}
			}
		}
		return nil, false
	}

	// Get specific section
	sectionVal, exists := om.Get(sectionSegment)
	if !exists {
		return nil, false
	}

	// If only one segment, return the whole section
	if len(segments) == 1 {
		return sectionVal, true
	}

	// Get key from section
	sectionMap := format.ToOrderedMapPtr(sectionVal)
	if sectionMap == nil {
		return nil, false
	}

	keySegment := segments[1]

	// Handle wildcard for key
	if keySegment == "*" {
		// Return first key value
		for _, keyName := range sectionMap.Keys() {
			val, _ := sectionMap.Get(keyName)
			return val, true
		}
		return nil, false
	}

	val, exists := sectionMap.Get(keySegment)
	return val, exists
}


// SetPath sets a value at the given path, supporting wildcards.
// INI paths are limited to ["section", "key"] format (max 2 segments).
// Values are converted to strings (INI only supports strings).
func (h *Handler) SetPath(tree any, p path.Path, value any) error {
	segments := p.Segments()
	if len(segments) == 0 || len(segments) > 2 {
		return fmt.Errorf("INI paths must have 1 or 2 segments, got %d", len(segments))
	}

	om := format.ToOrderedMapPtr(tree)
	if om == nil {
		return fmt.Errorf("tree is not an ordered map")
	}

	sectionSegment := segments[0]

	// Handle wildcard for section
	if sectionSegment == "*" {
		for _, sectionName := range om.Keys() {
			sectionVal, _ := om.Get(sectionName)
			sectionMap := format.ToOrderedMapPtr(sectionVal)
			if sectionMap == nil {
				continue
			}
			if len(segments) == 1 {
				// Replace entire section - convert value to string map
				om.Set(sectionName, value)
			} else {
				keySegment := segments[1]
				if keySegment == "*" {
					// Set all keys in section
					strVal := toString(value)
					for _, keyName := range sectionMap.Keys() {
						sectionMap.Set(keyName, strVal)
					}
				} else {
					sectionMap.Set(keySegment, toString(value))
				}
			}
		}
		return nil
	}

	// Get or create section
	sectionVal, exists := om.Get(sectionSegment)
	var sectionMap *orderedmap.OrderedMap
	if exists {
		sectionMap = format.ToOrderedMapPtr(sectionVal)
		if sectionMap == nil {
			return fmt.Errorf("section %q is not a map", sectionSegment)
		}
	} else {
		sectionMap = orderedmap.New()
		om.Set(sectionSegment, sectionMap)
	}

	// If only one segment, replace the whole section
	if len(segments) == 1 {
		om.Set(sectionSegment, value)
		return nil
	}

	keySegment := segments[1]

	// Handle wildcard for key
	if keySegment == "*" {
		strVal := toString(value)
		for _, keyName := range sectionMap.Keys() {
			sectionMap.Set(keyName, strVal)
		}
		return nil
	}

	// Set key in section (convert to string)
	sectionMap.Set(keySegment, toString(value))
	return nil
}

// Ensure Handler implements format.Handler.
var _ format.Handler = (*Handler)(nil)
