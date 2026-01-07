package merge

import (
	"testing"

	"github.com/iancoleman/orderedmap"
	"github.com/thirteen37/chezmoi-split/internal/format/json"
	"github.com/thirteen37/chezmoi-split/internal/path"
)

// Helper to create an ordered map from key-value pairs
func om(pairs ...any) *orderedmap.OrderedMap {
	m := orderedmap.New()
	for i := 0; i < len(pairs); i += 2 {
		m.Set(pairs[i].(string), pairs[i+1])
	}
	return m
}

func TestMerge(t *testing.T) {
	handler := json.New()

	tests := []struct {
		name    string
		managed *orderedmap.OrderedMap
		current *orderedmap.OrderedMap
		paths   [][]string
		check   func(result any) bool
	}{
		{
			name:    "no current config",
			managed: om("key", "managed"),
			current: nil,
			paths:   [][]string{{"key"}},
			check: func(result any) bool {
				r := result.(*orderedmap.OrderedMap)
				val, _ := r.Get("key")
				return val == "managed"
			},
		},
		{
			name:    "preserve app-owned path",
			managed: om("key", "managed", "app", "managed-value"),
			current: om("key", "current", "app", "current-value"),
			paths:   [][]string{{"app"}},
			check: func(result any) bool {
				r := result.(*orderedmap.OrderedMap)
				key, _ := r.Get("key")
				app, _ := r.Get("app")
				return key == "managed" && app == "current-value"
			},
		},
		{
			name:    "nested path preservation",
			managed: om("outer", om("inner", "managed")),
			current: om("outer", om("inner", "current")),
			paths:   [][]string{{"outer", "inner"}},
			check: func(result any) bool {
				r := result.(*orderedmap.OrderedMap)
				outer, _ := r.Get("outer")
				inner, _ := outer.(*orderedmap.OrderedMap).Get("inner")
				return inner == "current"
			},
		},
		{
			name:    "path not in current",
			managed: om("key", "managed", "app", "managed-default"),
			current: om("key", "current"),
			paths:   [][]string{{"app"}},
			check: func(result any) bool {
				r := result.(*orderedmap.OrderedMap)
				key, _ := r.Get("key")
				app, _ := r.Get("app")
				return key == "managed" && app == "managed-default"
			},
		},
		{
			name:    "multiple paths",
			managed: om("setting1", "managed1", "setting2", "managed2", "setting3", "managed3"),
			current: om("setting1", "current1", "setting2", "current2", "setting3", "current3"),
			paths:   [][]string{{"setting1"}, {"setting3"}},
			check: func(result any) bool {
				r := result.(*orderedmap.OrderedMap)
				s1, _ := r.Get("setting1")
				s2, _ := r.Get("setting2")
				s3, _ := r.Get("setting3")
				return s1 == "current1" && s2 == "managed2" && s3 == "current3"
			},
		},
		{
			name: "complex nested structure",
			managed: om("agent", om(
				"default_model", om("provider", "managed-provider", "model", "managed-model"),
				"profiles", om("ask", "managed-ask"),
			)),
			current: om("agent", om(
				"default_model", om("provider", "user-provider", "model", "user-model"),
				"profiles", om("ask", "user-ask"),
			)),
			paths: [][]string{{"agent", "default_model"}},
			check: func(result any) bool {
				r := result.(*orderedmap.OrderedMap)
				agent, _ := r.Get("agent")
				agentMap := agent.(*orderedmap.OrderedMap)
				dm, _ := agentMap.Get("default_model")
				dmMap := dm.(*orderedmap.OrderedMap)
				provider, _ := dmMap.Get("provider")
				model, _ := dmMap.Get("model")
				profiles, _ := agentMap.Get("profiles")
				profilesMap := profiles.(*orderedmap.OrderedMap)
				ask, _ := profilesMap.Get("ask")
				return provider == "user-provider" && model == "user-model" && ask == "managed-ask"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := make([]path.Path, len(tt.paths))
			for i, p := range tt.paths {
				paths[i] = path.NewArrayPath(p)
			}

			got := Merge(handler, tt.managed, tt.current, paths)
			if !tt.check(got) {
				t.Errorf("Merge() check failed for %s", tt.name)
			}
		})
	}
}

func TestMerge_DoesNotModifyOriginal(t *testing.T) {
	handler := json.New()

	managed := om("key", "managed")
	current := om("key", "current")
	paths := []path.Path{path.NewArrayPath([]string{"key"})}

	Merge(handler, managed, current, paths)

	// Original managed should be unchanged
	val, _ := managed.Get("key")
	if val != "managed" {
		t.Errorf("Merge() modified original managed map")
	}
}

func TestMerge_PreservesOrder(t *testing.T) {
	handler := json.New()

	// Managed has specific order: zebra, apple, mango
	managed := om("zebra", "z", "apple", "a", "mango", "m")
	current := om("zebra", "z2", "apple", "a2", "mango", "m2")

	// Ignore apple (should come from current)
	paths := []path.Path{path.NewArrayPath([]string{"apple"})}

	got := Merge(handler, managed, current, paths)
	result := got.(*orderedmap.OrderedMap)

	// Check that order is preserved: zebra, apple, mango
	keys := result.Keys()
	expectedKeys := []string{"zebra", "apple", "mango"}
	for i, k := range keys {
		if k != expectedKeys[i] {
			t.Errorf("Merge() key[%d] = %q, want %q", i, k, expectedKeys[i])
		}
	}

	// Check values
	apple, _ := result.Get("apple")
	if apple != "a2" {
		t.Errorf("Merge() apple = %v, want a2", apple)
	}
}
