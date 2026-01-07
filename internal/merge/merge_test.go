package merge

import (
	"reflect"
	"testing"

	"github.com/thirteen37/chezmoi-split/internal/format/json"
	"github.com/thirteen37/chezmoi-split/internal/path"
)

func TestMerge(t *testing.T) {
	handler := json.New()

	tests := []struct {
		name    string
		managed map[string]any
		current map[string]any
		paths   [][]string
		want    map[string]any
	}{
		{
			name:    "no current config",
			managed: map[string]any{"key": "managed"},
			current: nil,
			paths:   [][]string{{"key"}},
			want:    map[string]any{"key": "managed"},
		},
		{
			name:    "preserve app-owned path",
			managed: map[string]any{"key": "managed", "app": "managed-value"},
			current: map[string]any{"key": "current", "app": "current-value"},
			paths:   [][]string{{"app"}},
			want:    map[string]any{"key": "managed", "app": "current-value"},
		},
		{
			name:    "nested path preservation",
			managed: map[string]any{"outer": map[string]any{"inner": "managed"}},
			current: map[string]any{"outer": map[string]any{"inner": "current"}},
			paths:   [][]string{{"outer", "inner"}},
			want:    map[string]any{"outer": map[string]any{"inner": "current"}},
		},
		{
			name:    "path not in current",
			managed: map[string]any{"key": "managed", "app": "managed-default"},
			current: map[string]any{"key": "current"},
			paths:   [][]string{{"app"}},
			want:    map[string]any{"key": "managed", "app": "managed-default"},
		},
		{
			name: "multiple paths",
			managed: map[string]any{
				"setting1": "managed1",
				"setting2": "managed2",
				"setting3": "managed3",
			},
			current: map[string]any{
				"setting1": "current1",
				"setting2": "current2",
				"setting3": "current3",
			},
			paths: [][]string{{"setting1"}, {"setting3"}},
			want: map[string]any{
				"setting1": "current1",
				"setting2": "managed2",
				"setting3": "current3",
			},
		},
		{
			name: "complex nested structure",
			managed: map[string]any{
				"agent": map[string]any{
					"default_model": map[string]any{
						"provider": "managed-provider",
						"model":    "managed-model",
					},
					"profiles": map[string]any{
						"ask": "managed-ask",
					},
				},
			},
			current: map[string]any{
				"agent": map[string]any{
					"default_model": map[string]any{
						"provider": "user-provider",
						"model":    "user-model",
					},
					"profiles": map[string]any{
						"ask": "user-ask",
					},
				},
			},
			paths: [][]string{{"agent", "default_model"}},
			want: map[string]any{
				"agent": map[string]any{
					"default_model": map[string]any{
						"provider": "user-provider",
						"model":    "user-model",
					},
					"profiles": map[string]any{
						"ask": "managed-ask",
					},
				},
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
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Merge() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMerge_DoesNotModifyOriginal(t *testing.T) {
	handler := json.New()

	managed := map[string]any{"key": "managed"}
	current := map[string]any{"key": "current"}
	paths := []path.Path{path.NewArrayPath([]string{"key"})}

	Merge(handler, managed, current, paths)

	// Original managed should be unchanged
	if managed["key"] != "managed" {
		t.Errorf("Merge() modified original managed map")
	}
}
