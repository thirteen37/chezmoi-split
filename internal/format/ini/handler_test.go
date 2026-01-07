package ini

import (
	"strings"
	"testing"

	"github.com/iancoleman/orderedmap"
	"github.com/thirteen37/chezmoi-split/internal/format"
	"github.com/thirteen37/chezmoi-split/internal/path"
)

func TestHandler_Parse(t *testing.T) {
	h := New()

	tests := []struct {
		name     string
		input    string
		wantKeys []string
		wantErr  bool
	}{
		{
			name:     "simple section",
			input:    "[section]\nkey = value",
			wantKeys: []string{"section"},
		},
		{
			name:     "multiple sections",
			input:    "[section1]\nkey1 = value1\n\n[section2]\nkey2 = value2",
			wantKeys: []string{"section1", "section2"},
		},
		{
			name:     "empty ini",
			input:    "",
			wantKeys: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := h.Parse([]byte(tt.input), format.ParseOptions{})
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
					t.Errorf("Parse() got %d keys (%v), want %d (%v)", len(gotKeys), gotKeys, len(tt.wantKeys), tt.wantKeys)
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

func TestHandler_Parse_StripCommentsError(t *testing.T) {
	h := New()

	_, err := h.Parse([]byte("[section]\nkey = value"), format.ParseOptions{StripComments: true})
	if err == nil {
		t.Error("Parse() with StripComments should return error for INI")
	}
}

func TestHandler_Parse_Values(t *testing.T) {
	h := New()

	input := `[database]
host = localhost
port = 3306
enabled = true
`

	tree, err := h.Parse([]byte(input), format.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	om := tree.(*orderedmap.OrderedMap)
	db, exists := om.Get("database")
	if !exists {
		t.Fatal("Parse() missing 'database' section")
	}

	dbMap := db.(*orderedmap.OrderedMap)

	// All values should be strings in INI
	host, _ := dbMap.Get("host")
	if host != "localhost" {
		t.Errorf("host = %v, want 'localhost'", host)
	}

	port, _ := dbMap.Get("port")
	if port != "3306" {
		t.Errorf("port = %v, want '3306' (string)", port)
	}

	enabled, _ := dbMap.Get("enabled")
	if enabled != "true" {
		t.Errorf("enabled = %v, want 'true' (string)", enabled)
	}
}

func TestHandler_GetPath(t *testing.T) {
	h := New()

	// Build tree
	section := orderedmap.New()
	section.Set("key", "value")

	tree := orderedmap.New()
	tree.Set("section", section)

	tests := []struct {
		name      string
		path      []string
		wantVal   any
		wantFound bool
	}{
		{
			name:      "section only",
			path:      []string{"section"},
			wantVal:   section,
			wantFound: true,
		},
		{
			name:      "section.key",
			path:      []string{"section", "key"},
			wantVal:   "value",
			wantFound: true,
		},
		{
			name:      "non-existent section",
			path:      []string{"missing"},
			wantFound: false,
		},
		{
			name:      "non-existent key",
			path:      []string{"section", "missing"},
			wantFound: false,
		},
		{
			name:      "too deep path",
			path:      []string{"section", "key", "deep"},
			wantFound: false,
		},
		{
			name:      "empty path",
			path:      []string{},
			wantFound: false,
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

func TestHandler_GetPath_Wildcard(t *testing.T) {
	h := New()

	// Build tree with multiple sections
	section1 := orderedmap.New()
	section1.Set("host", "server1.example.com")
	section1.Set("port", "8080")

	section2 := orderedmap.New()
	section2.Set("host", "server2.example.com")
	section2.Set("port", "9090")

	tree := orderedmap.New()
	tree.Set("server1", section1)
	tree.Set("server2", section2)

	t.Run("wildcard section", func(t *testing.T) {
		p := path.NewArrayPath([]string{"*", "host"})
		got, found := h.GetPath(tree, p)
		if !found {
			t.Error("GetPath() with wildcard should find a match")
		}
		// Should return first match
		if got != "server1.example.com" {
			t.Errorf("GetPath() = %v, want 'server1.example.com'", got)
		}
	})

	t.Run("wildcard key", func(t *testing.T) {
		p := path.NewArrayPath([]string{"server1", "*"})
		got, found := h.GetPath(tree, p)
		if !found {
			t.Error("GetPath() with wildcard key should find a match")
		}
		// Should return first key value
		if got != "server1.example.com" {
			t.Errorf("GetPath() = %v, want 'server1.example.com'", got)
		}
	})
}

func TestHandler_SetPath(t *testing.T) {
	h := New()

	t.Run("set existing key", func(t *testing.T) {
		section := orderedmap.New()
		section.Set("key", "old")
		tree := orderedmap.New()
		tree.Set("section", section)

		p := path.NewArrayPath([]string{"section", "key"})
		err := h.SetPath(tree, p, "new")
		if err != nil {
			t.Errorf("SetPath() error = %v", err)
			return
		}

		got, _ := section.Get("key")
		if got != "new" {
			t.Errorf("SetPath() key = %v, want new", got)
		}
	})

	t.Run("create new section", func(t *testing.T) {
		tree := orderedmap.New()

		p := path.NewArrayPath([]string{"newsection", "key"})
		err := h.SetPath(tree, p, "value")
		if err != nil {
			t.Errorf("SetPath() error = %v", err)
			return
		}

		section, exists := tree.Get("newsection")
		if !exists {
			t.Error("SetPath() did not create section")
			return
		}

		sectionMap := section.(*orderedmap.OrderedMap)
		got, _ := sectionMap.Get("key")
		if got != "value" {
			t.Errorf("SetPath() key = %v, want value", got)
		}
	})

	t.Run("reject too deep path", func(t *testing.T) {
		tree := orderedmap.New()

		p := path.NewArrayPath([]string{"a", "b", "c"})
		err := h.SetPath(tree, p, "value")
		if err == nil {
			t.Error("SetPath() should reject 3-segment path for INI")
		}
	})

	t.Run("reject empty path", func(t *testing.T) {
		tree := orderedmap.New()

		p := path.NewArrayPath([]string{})
		err := h.SetPath(tree, p, "value")
		if err == nil {
			t.Error("SetPath() should reject empty path")
		}
	})
}

func TestHandler_SetPath_TypeConversion(t *testing.T) {
	h := New()

	section := orderedmap.New()
	section.Set("key", "original")
	tree := orderedmap.New()
	tree.Set("section", section)

	t.Run("int to string", func(t *testing.T) {
		p := path.NewArrayPath([]string{"section", "key"})
		err := h.SetPath(tree, p, 42)
		if err != nil {
			t.Errorf("SetPath() error = %v", err)
		}
		got, _ := section.Get("key")
		if got != "42" {
			t.Errorf("SetPath() key = %v (%T), want '42' (string)", got, got)
		}
	})

	t.Run("bool to string", func(t *testing.T) {
		p := path.NewArrayPath([]string{"section", "key"})
		err := h.SetPath(tree, p, true)
		if err != nil {
			t.Errorf("SetPath() error = %v", err)
		}
		got, _ := section.Get("key")
		if got != "true" {
			t.Errorf("SetPath() key = %v (%T), want 'true' (string)", got, got)
		}
	})

	t.Run("float to string", func(t *testing.T) {
		p := path.NewArrayPath([]string{"section", "key"})
		err := h.SetPath(tree, p, 3.14)
		if err != nil {
			t.Errorf("SetPath() error = %v", err)
		}
		got, _ := section.Get("key")
		if got != "3.14" {
			t.Errorf("SetPath() key = %v (%T), want '3.14' (string)", got, got)
		}
	})
}

func TestHandler_SetPath_Wildcard(t *testing.T) {
	h := New()

	// Build tree with multiple sections
	section1 := orderedmap.New()
	section1.Set("enabled", "true")

	section2 := orderedmap.New()
	section2.Set("enabled", "true")

	tree := orderedmap.New()
	tree.Set("server1", section1)
	tree.Set("server2", section2)

	t.Run("wildcard section", func(t *testing.T) {
		p := path.NewArrayPath([]string{"*", "enabled"})
		err := h.SetPath(tree, p, false)
		if err != nil {
			t.Errorf("SetPath() error = %v", err)
		}

		// Both should now be "false" (converted to string)
		s1enabled, _ := section1.Get("enabled")
		s2enabled, _ := section2.Get("enabled")

		if s1enabled != "false" {
			t.Errorf("SetPath() server1.enabled = %v, want 'false'", s1enabled)
		}
		if s2enabled != "false" {
			t.Errorf("SetPath() server2.enabled = %v, want 'false'", s2enabled)
		}
	})
}

func TestHandler_Serialize(t *testing.T) {
	h := New()

	section := orderedmap.New()
	section.Set("key", "value")

	tree := orderedmap.New()
	tree.Set("section", section)

	data, err := h.Serialize(tree, format.SerializeOptions{})
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Should contain section header and key
	output := string(data)
	if !strings.Contains(output, "[section]") {
		t.Errorf("Serialize() missing section header: %q", output)
	}
	if !strings.Contains(output, "key") && !strings.Contains(output, "value") {
		t.Errorf("Serialize() missing key/value: %q", output)
	}
}

func TestHandler_ParseAndSerialize_RoundTrip(t *testing.T) {
	h := New()

	input := `[database]
host = localhost
port = 3306

[server]
address = 0.0.0.0
`

	tree, err := h.Parse([]byte(input), format.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Modify a value
	p := path.NewArrayPath([]string{"database", "port"})
	err = h.SetPath(tree, p, "5432")
	if err != nil {
		t.Fatalf("SetPath() error = %v", err)
	}

	// Serialize
	data, err := h.Serialize(tree, format.SerializeOptions{})
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Re-parse and verify
	tree2, err := h.Parse(data, format.ParseOptions{})
	if err != nil {
		t.Fatalf("Re-parse error = %v", err)
	}

	port, found := h.GetPath(tree2, p)
	if !found || port != "5432" {
		t.Errorf("Round-trip port = %v, want '5432'", port)
	}
}
