package toml

import (
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
			name:     "simple toml",
			input:    `key = "value"`,
			wantKeys: []string{"key"},
		},
		{
			name:     "with section",
			input:    "[section]\nkey = \"value\"",
			wantKeys: []string{"section"},
		},
		{
			name:     "nested section",
			input:    "[outer]\n[outer.inner]\nkey = \"value\"",
			wantKeys: []string{"outer"},
		},
		{
			name:    "invalid toml",
			input:   `[invalid`,
			wantErr: true,
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

	_, err := h.Parse([]byte(`key = "value"`), format.ParseOptions{StripComments: true})
	if err == nil {
		t.Error("Parse() with StripComments should return error for TOML")
	}
}

func TestHandler_Parse_PreservesOrder(t *testing.T) {
	h := New()

	// Keys in specific order: zebra, apple, mango
	input := `zebra = "z"
apple = "a"
mango = "m"
`

	tree, err := h.Parse([]byte(input), format.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	om := tree.(*orderedmap.OrderedMap)
	keys := om.Keys()
	expected := []string{"zebra", "apple", "mango"}

	if len(keys) != len(expected) {
		t.Fatalf("Parse() got %d keys, want %d", len(keys), len(expected))
	}

	for i, k := range keys {
		if k != expected[i] {
			t.Errorf("Parse() key[%d] = %q, want %q (order not preserved)", i, k, expected[i])
		}
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

func TestHandler_GetPath_Wildcard(t *testing.T) {
	h := New()

	// Build tree with multiple servers
	server1 := orderedmap.New()
	server1.Set("enabled", true)
	server1.Set("host", "server1.example.com")

	server2 := orderedmap.New()
	server2.Set("enabled", false)
	server2.Set("host", "server2.example.com")

	servers := orderedmap.New()
	servers.Set("server1", server1)
	servers.Set("server2", server2)

	tree := orderedmap.New()
	tree.Set("servers", servers)

	// Wildcard should find first match
	p := path.NewArrayPath([]string{"servers", "*", "enabled"})
	got, found := h.GetPath(tree, p)
	if !found {
		t.Error("GetPath() with wildcard should find a match")
	}
	if got != true {
		t.Errorf("GetPath() = %v, want true (first server)", got)
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

func TestHandler_SetPath_Wildcard(t *testing.T) {
	h := New()

	// Build tree with multiple servers
	server1 := orderedmap.New()
	server1.Set("enabled", true)

	server2 := orderedmap.New()
	server2.Set("enabled", true)

	servers := orderedmap.New()
	servers.Set("server1", server1)
	servers.Set("server2", server2)

	tree := orderedmap.New()
	tree.Set("servers", servers)

	// Set all servers to disabled using wildcard
	p := path.NewArrayPath([]string{"servers", "*", "enabled"})
	err := h.SetPath(tree, p, false)
	if err != nil {
		t.Errorf("SetPath() error = %v", err)
	}

	// Verify both are now false
	s1enabled, _ := server1.Get("enabled")
	s2enabled, _ := server2.Get("enabled")

	if s1enabled != false {
		t.Errorf("SetPath() server1.enabled = %v, want false", s1enabled)
	}
	if s2enabled != false {
		t.Errorf("SetPath() server2.enabled = %v, want false", s2enabled)
	}
}

func TestHandler_Serialize(t *testing.T) {
	h := New()

	tree := orderedmap.New()
	tree.Set("key", "value")

	data, err := h.Serialize(tree, format.SerializeOptions{})
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Should produce valid TOML
	want := "key = \"value\"\n"
	if string(data) != want {
		t.Errorf("Serialize() = %q, want %q", string(data), want)
	}
}

func TestHandler_ParseAndSerialize_RoundTrip(t *testing.T) {
	h := New()

	input := `[server]
host = "localhost"
port = 8080

[server.tls]
enabled = true
`

	tree, err := h.Parse([]byte(input), format.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Verify parsed structure
	om := tree.(*orderedmap.OrderedMap)
	server, exists := om.Get("server")
	if !exists {
		t.Fatal("Parse() missing 'server' key")
	}

	serverMap := server.(*orderedmap.OrderedMap)
	host, exists := serverMap.Get("host")
	if !exists || host != "localhost" {
		t.Errorf("Parse() server.host = %v, want 'localhost'", host)
	}

	// Test GetPath on parsed data
	p := path.NewArrayPath([]string{"server", "tls", "enabled"})
	enabled, found := h.GetPath(tree, p)
	if !found {
		t.Error("GetPath() server.tls.enabled not found")
	}
	if enabled != true {
		t.Errorf("GetPath() server.tls.enabled = %v, want true", enabled)
	}

	// Serialize back (order may differ due to encoder)
	data, err := h.Serialize(tree, format.SerializeOptions{})
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Should be valid TOML that can be re-parsed
	_, err = h.Parse(data, format.ParseOptions{})
	if err != nil {
		t.Errorf("Re-parse serialized data error = %v", err)
	}
}

func TestHandler_ParseWithTypes(t *testing.T) {
	h := New()

	input := `
string = "hello"
integer = 42
float = 3.14
boolean = true
array = [1, 2, 3]
`

	tree, err := h.Parse([]byte(input), format.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	om := tree.(*orderedmap.OrderedMap)

	// Check types are preserved
	str, _ := om.Get("string")
	if str != "hello" {
		t.Errorf("string = %v, want 'hello'", str)
	}

	integer, _ := om.Get("integer")
	if integer != int64(42) {
		t.Errorf("integer = %v (%T), want 42", integer, integer)
	}

	float, _ := om.Get("float")
	if float != 3.14 {
		t.Errorf("float = %v, want 3.14", float)
	}

	boolean, _ := om.Get("boolean")
	if boolean != true {
		t.Errorf("boolean = %v, want true", boolean)
	}

	arr, _ := om.Get("array")
	arrSlice, ok := arr.([]any)
	if !ok || len(arrSlice) != 3 {
		t.Errorf("array = %v (%T), want [1, 2, 3]", arr, arr)
	}
}
