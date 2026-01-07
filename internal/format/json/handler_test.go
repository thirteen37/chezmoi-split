package json

import (
	"testing"

	"github.com/iancoleman/orderedmap"
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
		name     string
		input    string
		opts     format.ParseOptions
		wantKeys []string
		wantErr  bool
	}{
		{
			name:     "simple json",
			input:    `{"key": "value"}`,
			wantKeys: []string{"key"},
		},
		{
			name:     "nested json",
			input:    `{"outer": {"inner": "value"}}`,
			wantKeys: []string{"outer"},
		},
		{
			name:     "json with comments stripped",
			input:    "// comment\n{\"key\": \"value\"}",
			opts:     format.ParseOptions{StripComments: true},
			wantKeys: []string{"key"},
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
			if !tt.wantErr {
				om, ok := got.(*orderedmap.OrderedMap)
				if !ok {
					t.Errorf("Parse() returned %T, want *orderedmap.OrderedMap", got)
					return
				}
				gotKeys := om.Keys()
				if len(gotKeys) != len(tt.wantKeys) {
					t.Errorf("Parse() got %d keys, want %d", len(gotKeys), len(tt.wantKeys))
					return
				}
				for i, k := range gotKeys {
					if k != tt.wantKeys[i] {
						t.Errorf("Parse() key[%d] = %q, want %q", i, k, tt.wantKeys[i])
					}
				}
			}
		})
	}
}

func TestHandler_GetPath(t *testing.T) {
	h := New()

	// Build ordered map tree
	level2 := orderedmap.New()
	level2.Set("value", "found")

	level1 := orderedmap.New()
	level1.Set("level2", level2)

	tree := orderedmap.New()
	tree.Set("level1", level1)
	tree.Set("simple", "direct")

	tests := []struct {
		name      string
		path      []string
		wantVal   any
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
			wantVal:   level2,
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
			if tt.wantFound && got != tt.wantVal {
				t.Errorf("GetPath() = %v, want %v", got, tt.wantVal)
			}
		})
	}
}

func TestHandler_SetPath(t *testing.T) {
	h := New()

	t.Run("set existing path", func(t *testing.T) {
		tree := orderedmap.New()
		tree.Set("key", "old")

		p := path.NewArrayPath([]string{"key"})
		err := h.SetPath(tree, p, "new")
		if err != nil {
			t.Errorf("SetPath() error = %v", err)
			return
		}

		got, _ := tree.Get("key")
		if got != "new" {
			t.Errorf("SetPath() key = %v, want new", got)
		}
	})

	t.Run("set nested path", func(t *testing.T) {
		inner := orderedmap.New()
		inner.Set("inner", "old")
		tree := orderedmap.New()
		tree.Set("outer", inner)

		p := path.NewArrayPath([]string{"outer", "inner"})
		err := h.SetPath(tree, p, "new")
		if err != nil {
			t.Errorf("SetPath() error = %v", err)
			return
		}

		got, _ := inner.Get("inner")
		if got != "new" {
			t.Errorf("SetPath() inner = %v, want new", got)
		}
	})

	t.Run("create intermediate maps", func(t *testing.T) {
		tree := orderedmap.New()

		p := path.NewArrayPath([]string{"a", "b", "c"})
		err := h.SetPath(tree, p, "deep")
		if err != nil {
			t.Errorf("SetPath() error = %v", err)
			return
		}

		a, _ := tree.Get("a")
		aMap := a.(*orderedmap.OrderedMap)
		b, _ := aMap.Get("b")
		bMap := b.(*orderedmap.OrderedMap)
		c, _ := bMap.Get("c")
		if c != "deep" {
			t.Errorf("SetPath() deep value = %v, want deep", c)
		}
	})
}

func TestHandler_Serialize_PreservesOrder(t *testing.T) {
	h := New()

	// Create ordered map with specific key order
	tree := orderedmap.New()
	tree.Set("zebra", "last")
	tree.Set("apple", "first")
	tree.Set("mango", "middle")

	data, err := h.Serialize(tree, format.SerializeOptions{})
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// The order should be zebra, apple, mango (insertion order)
	want := "{\n  \"zebra\": \"last\",\n  \"apple\": \"first\",\n  \"mango\": \"middle\"\n}\n"
	if string(data) != want {
		t.Errorf("Serialize() = %q, want %q", string(data), want)
	}
}

func TestHandler_ParseAndSerialize_PreservesOrder(t *testing.T) {
	h := New()

	// Parse JSON with specific key order
	input := `{"zebra": "last", "apple": "first", "mango": "middle"}`

	tree, err := h.Parse([]byte(input), format.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	data, err := h.Serialize(tree, format.SerializeOptions{})
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// The order should be preserved: zebra, apple, mango
	want := "{\n  \"zebra\": \"last\",\n  \"apple\": \"first\",\n  \"mango\": \"middle\"\n}\n"
	if string(data) != want {
		t.Errorf("ParseAndSerialize() = %q, want %q", string(data), want)
	}
}
