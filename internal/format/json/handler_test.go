package json

import (
	"reflect"
	"testing"

	"github.com/thirteen37/chezmoi-split/internal/format"
	"github.com/thirteen37/chezmoi-split/internal/path"
)

func TestStripComments(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no comments",
			input: `{"key": "value"}`,
			want:  `{"key": "value"}`,
		},
		{
			name:  "single line comment",
			input: "// comment\n{\"key\": \"value\"}",
			want:  "\n{\"key\": \"value\"}",
		},
		{
			name:  "inline comment",
			input: "{\"key\": \"value\"} // comment",
			want:  "{\"key\": \"value\"} ",
		},
		{
			name:  "comment with leading whitespace",
			input: "  // comment\n{\"key\": \"value\"}",
			want:  "\n{\"key\": \"value\"}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(StripComments([]byte(tt.input)))
			if got != tt.want {
				t.Errorf("StripComments() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHandler_Parse(t *testing.T) {
	h := New()

	tests := []struct {
		name    string
		input   string
		opts    format.ParseOptions
		want    map[string]any
		wantErr bool
	}{
		{
			name:  "simple json",
			input: `{"key": "value"}`,
			want:  map[string]any{"key": "value"},
		},
		{
			name:  "nested json",
			input: `{"outer": {"inner": "value"}}`,
			want:  map[string]any{"outer": map[string]any{"inner": "value"}},
		},
		{
			name:  "json with comments stripped",
			input: "// comment\n{\"key\": \"value\"}",
			opts:  format.ParseOptions{StripComments: true},
			want:  map[string]any{"key": "value"},
		},
		{
			name:    "invalid json",
			input:   `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := h.Parse([]byte(tt.input), tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandler_GetPath(t *testing.T) {
	h := New()
	tree := map[string]any{
		"level1": map[string]any{
			"level2": map[string]any{
				"value": "found",
			},
		},
		"simple": "direct",
	}

	tests := []struct {
		name     string
		path     []string
		wantVal  any
		wantFound bool
	}{
		{
			name:      "simple path",
			path:      []string{"simple"},
			wantVal:   "direct",
			wantFound: true,
		},
		{
			name:      "nested path",
			path:      []string{"level1", "level2", "value"},
			wantVal:   "found",
			wantFound: true,
		},
		{
			name:      "non-existent path",
			path:      []string{"missing"},
			wantFound: false,
		},
		{
			name:      "partial path to map",
			path:      []string{"level1", "level2"},
			wantVal:   map[string]any{"value": "found"},
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := path.NewArrayPath(tt.path)
			got, found := h.GetPath(tree, p)
			if found != tt.wantFound {
				t.Errorf("GetPath() found = %v, want %v", found, tt.wantFound)
			}
			if tt.wantFound && !reflect.DeepEqual(got, tt.wantVal) {
				t.Errorf("GetPath() = %v, want %v", got, tt.wantVal)
			}
		})
	}
}

func TestHandler_SetPath(t *testing.T) {
	h := New()

	tests := []struct {
		name    string
		tree    map[string]any
		path    []string
		value   any
		want    map[string]any
		wantErr bool
	}{
		{
			name:  "set existing path",
			tree:  map[string]any{"key": "old"},
			path:  []string{"key"},
			value: "new",
			want:  map[string]any{"key": "new"},
		},
		{
			name:  "set nested path",
			tree:  map[string]any{"outer": map[string]any{"inner": "old"}},
			path:  []string{"outer", "inner"},
			value: "new",
			want:  map[string]any{"outer": map[string]any{"inner": "new"}},
		},
		{
			name:  "create intermediate maps",
			tree:  map[string]any{},
			path:  []string{"a", "b", "c"},
			value: "deep",
			want:  map[string]any{"a": map[string]any{"b": map[string]any{"c": "deep"}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := path.NewArrayPath(tt.path)
			err := h.SetPath(tt.tree, p, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(tt.tree, tt.want) {
				t.Errorf("SetPath() tree = %v, want %v", tt.tree, tt.want)
			}
		})
	}
}
