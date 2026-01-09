package plaintext

import (
	"strings"
	"testing"

	"github.com/thirteen37/chezmoi-split/internal/format"
	"github.com/thirteen37/chezmoi-split/internal/path"
)

func TestHandler_Parse_WithMarkers(t *testing.T) {
	h := New()

	input := `# chezmoi:managed
set number
set expandtab

# chezmoi:ignored
colorscheme gruvbox

# chezmoi:end
`

	tree, err := h.Parse([]byte(input), format.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	config := tree.(*ParsedConfig)

	if len(config.Blocks) != 2 {
		t.Errorf("Parse() got %d blocks, want 2", len(config.Blocks))
		return
	}

	// First block should be managed
	if config.Blocks[0].Type != BlockManaged {
		t.Errorf("Block 0 type = %v, want BlockManaged", config.Blocks[0].Type)
	}
	if len(config.Blocks[0].Lines) != 3 { // includes blank line
		t.Errorf("Block 0 has %d lines, want 3", len(config.Blocks[0].Lines))
	}

	// Second block should be ignored
	if config.Blocks[1].Type != BlockIgnored {
		t.Errorf("Block 1 type = %v, want BlockIgnored", config.Blocks[1].Type)
	}
}

func TestHandler_Parse_NoMarkers(t *testing.T) {
	h := New()

	input := `set number
set expandtab
colorscheme gruvbox
`

	tree, err := h.Parse([]byte(input), format.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	config := tree.(*ParsedConfig)

	// Should have one implicit ignored block
	if len(config.Blocks) != 1 {
		t.Errorf("Parse() got %d blocks, want 1", len(config.Blocks))
		return
	}

	if config.Blocks[0].Type != BlockIgnored {
		t.Errorf("Block 0 type = %v, want BlockIgnored (implicit)", config.Blocks[0].Type)
	}

	if config.Blocks[0].MarkerLine != "" {
		t.Errorf("Block 0 should have no marker line for implicit block")
	}
}

func TestHandler_Parse_MarkerAnywhereInLine(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		wantType string
	}{
		{"simple", "# chezmoi:managed", "managed"},
		{"double hash", "## chezmoi:managed", "managed"},
		{"decorated", "# --- chezmoi:managed ---", "managed"},
		{"with padding", "   # chezmoi:ignored   ", "ignored"},
		{"end marker", "# chezmoi:end", "end"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectMarker(tt.line)
			if got != tt.wantType {
				t.Errorf("detectMarker(%q) = %q, want %q", tt.line, got, tt.wantType)
			}
		})
	}
}

func TestHandler_Serialize(t *testing.T) {
	h := New()

	config := &ParsedConfig{
		Blocks: []Block{
			{
				Type:       BlockManaged,
				MarkerLine: "# chezmoi:managed",
				Lines:      []string{"set number", "set expandtab"},
			},
			{
				Type:       BlockIgnored,
				MarkerLine: "# chezmoi:ignored",
				Lines:      []string{"colorscheme gruvbox"},
			},
		},
	}

	data, err := h.Serialize(config, format.SerializeOptions{})
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	output := string(data)

	if !strings.Contains(output, "chezmoi:managed") {
		t.Errorf("Serialize() missing managed marker: %q", output)
	}
	if !strings.Contains(output, "set number") {
		t.Errorf("Serialize() missing 'set number': %q", output)
	}
	if !strings.Contains(output, "chezmoi:ignored") {
		t.Errorf("Serialize() missing ignored marker: %q", output)
	}
	if !strings.Contains(output, "colorscheme gruvbox") {
		t.Errorf("Serialize() missing 'colorscheme gruvbox': %q", output)
	}
	if !strings.Contains(output, "chezmoi:end") {
		t.Errorf("Serialize() missing end marker: %q", output)
	}
}

func TestHandler_MergeBlocks_Basic(t *testing.T) {
	h := New()

	managed := &ParsedConfig{
		Blocks: []Block{
			{
				Type:       BlockManaged,
				MarkerLine: "# chezmoi:managed",
				Lines:      []string{"managed-line-1", "managed-line-2"},
			},
			{
				Type:       BlockIgnored,
				MarkerLine: "# chezmoi:ignored",
				Lines:      []string{"default-ignored"},
			},
		},
	}

	current := &ParsedConfig{
		Blocks: []Block{
			{
				Type:       BlockManaged,
				MarkerLine: "# chezmoi:managed",
				Lines:      []string{"old-managed"},
			},
			{
				Type:       BlockIgnored,
				MarkerLine: "# chezmoi:ignored",
				Lines:      []string{"user-preference"},
			},
		},
	}

	result := h.MergeBlocks(managed, current)

	if len(result.Blocks) != 2 {
		t.Fatalf("MergeBlocks() got %d blocks, want 2", len(result.Blocks))
	}

	// Managed block should have content from managed template
	if len(result.Blocks[0].Lines) != 2 || result.Blocks[0].Lines[0] != "managed-line-1" {
		t.Errorf("Managed block should have template content, got: %v", result.Blocks[0].Lines)
	}

	// Ignored block should have content from current
	if len(result.Blocks[1].Lines) != 1 || result.Blocks[1].Lines[0] != "user-preference" {
		t.Errorf("Ignored block should have current content, got: %v", result.Blocks[1].Lines)
	}
}

func TestHandler_MergeBlocks_CurrentNoMarkers(t *testing.T) {
	h := New()

	managed := &ParsedConfig{
		Blocks: []Block{
			{
				Type:       BlockManaged,
				MarkerLine: "# chezmoi:managed",
				Lines:      []string{"managed-line"},
			},
			{
				Type:       BlockIgnored,
				MarkerLine: "# chezmoi:ignored",
				Lines:      []string{"default"},
			},
		},
	}

	// Current has no markers (implicit ignored block)
	current := &ParsedConfig{
		Blocks: []Block{
			{
				Type:       BlockIgnored,
				MarkerLine: "", // implicit
				Lines:      []string{"user-line-1", "user-line-2"},
			},
		},
	}

	result := h.MergeBlocks(managed, current)

	// Ignored block should have content from current's implicit block
	if len(result.Blocks[1].Lines) != 2 {
		t.Errorf("Ignored block should have 2 lines from current, got: %v", result.Blocks[1].Lines)
	}
}

func TestHandler_MergeBlocks_MissingIgnoredInCurrent(t *testing.T) {
	h := New()

	managed := &ParsedConfig{
		Blocks: []Block{
			{
				Type:       BlockManaged,
				MarkerLine: "# chezmoi:managed",
				Lines:      []string{"managed"},
			},
			{
				Type:       BlockIgnored,
				MarkerLine: "# chezmoi:ignored",
				Lines:      []string{"default-value"},
			},
		},
	}

	// Current is nil (empty file)
	result := h.MergeBlocks(managed, nil)

	// Ignored block should use default from managed
	if len(result.Blocks[1].Lines) != 1 || result.Blocks[1].Lines[0] != "default-value" {
		t.Errorf("Ignored block should use default, got: %v", result.Blocks[1].Lines)
	}
}

func TestHandler_MergeBlocks_MultipleIgnored(t *testing.T) {
	h := New()

	managed := &ParsedConfig{
		Blocks: []Block{
			{Type: BlockManaged, MarkerLine: "# chezmoi:managed", Lines: []string{"m1"}},
			{Type: BlockIgnored, MarkerLine: "# chezmoi:ignored", Lines: []string{"default1"}},
			{Type: BlockManaged, MarkerLine: "# chezmoi:managed", Lines: []string{"m2"}},
			{Type: BlockIgnored, MarkerLine: "# chezmoi:ignored", Lines: []string{"default2"}},
		},
	}

	current := &ParsedConfig{
		Blocks: []Block{
			{Type: BlockManaged, MarkerLine: "# chezmoi:managed", Lines: []string{"old-m1"}},
			{Type: BlockIgnored, MarkerLine: "# chezmoi:ignored", Lines: []string{"user1"}},
			{Type: BlockManaged, MarkerLine: "# chezmoi:managed", Lines: []string{"old-m2"}},
			{Type: BlockIgnored, MarkerLine: "# chezmoi:ignored", Lines: []string{"user2"}},
		},
	}

	result := h.MergeBlocks(managed, current)

	// Check managed blocks have template content
	if result.Blocks[0].Lines[0] != "m1" {
		t.Errorf("Block 0 should be 'm1', got %v", result.Blocks[0].Lines)
	}
	if result.Blocks[2].Lines[0] != "m2" {
		t.Errorf("Block 2 should be 'm2', got %v", result.Blocks[2].Lines)
	}

	// Check ignored blocks match by index
	if result.Blocks[1].Lines[0] != "user1" {
		t.Errorf("Block 1 should be 'user1', got %v", result.Blocks[1].Lines)
	}
	if result.Blocks[3].Lines[0] != "user2" {
		t.Errorf("Block 3 should be 'user2', got %v", result.Blocks[3].Lines)
	}
}

func TestHandler_GetPath_NotSupported(t *testing.T) {
	h := New()

	config := &ParsedConfig{}
	p := path.NewArrayPath([]string{"anything"})

	_, found := h.GetPath(config, p)
	if found {
		t.Error("GetPath() should always return false for plaintext")
	}
}

func TestHandler_SetPath_NotSupported(t *testing.T) {
	h := New()

	config := &ParsedConfig{}
	p := path.NewArrayPath([]string{"anything"})

	err := h.SetPath(config, p, "value")
	if err == nil {
		t.Error("SetPath() should return error for plaintext")
	}
}

func TestHandler_RoundTrip(t *testing.T) {
	h := New()

	input := `# chezmoi:managed
set number
set expandtab

# chezmoi:ignored
colorscheme gruvbox

# chezmoi:end
`

	// Parse
	tree, err := h.Parse([]byte(input), format.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Serialize
	output, err := h.Serialize(tree, format.SerializeOptions{})
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Re-parse
	tree2, err := h.Parse(output, format.ParseOptions{})
	if err != nil {
		t.Fatalf("Re-parse error = %v", err)
	}

	config := tree2.(*ParsedConfig)

	// Verify structure preserved
	if len(config.Blocks) != 2 {
		t.Errorf("Round-trip got %d blocks, want 2", len(config.Blocks))
	}
}

func TestHandler_ContentBeforeFirstMarker_AddsEndMarker(t *testing.T) {
	// Regression test for Issue #6: hasExplicitMarkers heuristic bug
	// When content appears before first marker, the end marker should still be added
	h := New()

	input := `implicit content at start
# chezmoi:managed
managed content
# chezmoi:end
`

	tree, err := h.Parse([]byte(input), format.ParseOptions{})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	output, err := h.Serialize(tree, format.SerializeOptions{})
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	outputStr := string(output)

	// The output should contain the end marker
	if !strings.Contains(outputStr, "# chezmoi:end") {
		t.Errorf("Serialize() missing end marker, got:\n%s", outputStr)
	}

	// Should have both the implicit content and the managed block marker
	if !strings.Contains(outputStr, "implicit content at start") {
		t.Errorf("Serialize() missing implicit content")
	}
	if !strings.Contains(outputStr, "# chezmoi:managed") {
		t.Errorf("Serialize() missing managed marker")
	}
}

func TestHandler_MixedMarkers_NoSilentGeneration(t *testing.T) {
	// Regression test for Issue #2: we should never silently generate markers for blocks
	// If a block doesn't have a MarkerLine, it should be serialized without one
	h := New()

	// Programmatically create a mixed state
	// (first block has MarkerLine, second doesn't)
	config := &ParsedConfig{
		Blocks: []Block{
			{
				Type:       BlockManaged,
				MarkerLine: "# chezmoi:managed",
				Lines:      []string{"explicit content"},
			},
			{
				Type:       BlockIgnored,
				MarkerLine: "", // No marker
				Lines:      []string{"implicit content"},
			},
		},
	}

	output, err := h.Serialize(config, format.SerializeOptions{})
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	outputStr := string(output)

	// Should have the explicit marker from block 1
	if !strings.Contains(outputStr, "# chezmoi:managed") {
		t.Errorf("Missing explicit marker from first block")
	}

	// Should NOT generate a marker for block 2 - implicit content should appear without marker
	// Count how many times "chezmoi:" appears - should be 2 (managed + end), not 3
	count := strings.Count(outputStr, "chezmoi:")
	if count != 2 {
		t.Errorf("Found %d chezmoi markers, want 2 (managed + end, no generated ignored marker), output:\n%s", count, outputStr)
	}

	// Should have both contents
	if !strings.Contains(outputStr, "explicit content") {
		t.Errorf("Missing explicit content")
	}
	if !strings.Contains(outputStr, "implicit content") {
		t.Errorf("Missing implicit content")
	}

	// Should have end marker since we have explicit markers
	if !strings.Contains(outputStr, "# chezmoi:end") {
		t.Errorf("Missing end marker")
	}
}
