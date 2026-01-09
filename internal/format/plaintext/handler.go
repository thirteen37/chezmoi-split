// Package plaintext provides a handler for line-based plaintext config files.
package plaintext

import (
	"fmt"
	"strings"

	"github.com/thirteen37/chezmoi-split/internal/format"
	"github.com/thirteen37/chezmoi-split/internal/path"
)

// BlockType indicates the type of a block in a plaintext config.
type BlockType int

const (
	// BlockManaged indicates content controlled by chezmoi (from template).
	BlockManaged BlockType = iota
	// BlockIgnored indicates content preserved from current config (app/user-managed).
	BlockIgnored
	// BlockEnd is used internally when generating end markers.
	BlockEnd BlockType = -1
)

// Block represents a section of the config file.
type Block struct {
	Type       BlockType
	Lines      []string
	MarkerLine string // The original marker line (preserved for output)
}

// ParsedConfig holds the structured representation of a plaintext config.
type ParsedConfig struct {
	Blocks        []Block
	CommentPrefix string // Used when writing markers
	TrailingLines []string // Lines after the last chezmoi:end marker
}

// Handler implements format.Handler for plaintext files.
type Handler struct {
	CommentPrefix string
}

// New creates a new plaintext handler with the given comment prefix.
func New(commentPrefix string) *Handler {
	return &Handler{CommentPrefix: commentPrefix}
}

// Parse reads plaintext bytes and returns a *ParsedConfig.
// It scans for chezmoi:managed, chezmoi:ignored, and chezmoi:end markers anywhere in lines.
//
// NOTE: Marker detection is substring-based. If your config contains the literal
// string "chezmoi:managed" as data (e.g., in a comment about chezmoi-split),
// it will be incorrectly treated as a marker. There is no escaping mechanism.
func (h *Handler) Parse(data []byte, opts format.ParseOptions) (any, error) {
	lines := strings.Split(string(data), "\n")
	config := &ParsedConfig{
		CommentPrefix: h.CommentPrefix,
	}

	var currentBlock *Block
	afterEnd := false

	for _, line := range lines {
		markerType := detectMarker(line)

		switch markerType {
		case "managed":
			if currentBlock != nil {
				config.Blocks = append(config.Blocks, *currentBlock)
			}
			currentBlock = &Block{
				Type:       BlockManaged,
				MarkerLine: line,
			}
			afterEnd = false

		case "ignored":
			if currentBlock != nil {
				config.Blocks = append(config.Blocks, *currentBlock)
			}
			currentBlock = &Block{
				Type:       BlockIgnored,
				MarkerLine: line,
			}
			afterEnd = false

		case "end":
			if currentBlock != nil {
				config.Blocks = append(config.Blocks, *currentBlock)
				currentBlock = nil
			}
			afterEnd = true

		default:
			// Regular content line
			if afterEnd {
				config.TrailingLines = append(config.TrailingLines, line)
			} else if currentBlock != nil {
				currentBlock.Lines = append(currentBlock.Lines, line)
			} else {
				// Content before any marker - treat as implicit ignored block
				currentBlock = &Block{
					Type: BlockIgnored,
				}
				currentBlock.Lines = append(currentBlock.Lines, line)
			}
		}
	}

	// Close any open block
	if currentBlock != nil {
		config.Blocks = append(config.Blocks, *currentBlock)
	}

	return config, nil
}

// detectMarker checks if a line contains a chezmoi marker and returns its type.
// Returns "managed", "ignored", "end", or "" for no marker.
func detectMarker(line string) string {
	if strings.Contains(line, "chezmoi:managed") {
		return "managed"
	}
	if strings.Contains(line, "chezmoi:ignored") {
		return "ignored"
	}
	if strings.Contains(line, "chezmoi:end") {
		return "end"
	}
	return ""
}

// Serialize writes the ParsedConfig back to bytes.
func (h *Handler) Serialize(tree any, opts format.SerializeOptions) ([]byte, error) {
	config, ok := tree.(*ParsedConfig)
	if !ok {
		return nil, fmt.Errorf("tree is not a *ParsedConfig")
	}

	var lines []string
	hasExplicitMarkers := len(config.Blocks) > 0 && config.Blocks[0].MarkerLine != ""

	for _, block := range config.Blocks {
		// Add marker line
		if block.MarkerLine != "" {
			lines = append(lines, block.MarkerLine)
		} else if hasExplicitMarkers {
			// Generate marker for blocks that need one
			lines = append(lines, h.generateMarker(block.Type))
		}
		// Add content lines
		lines = append(lines, block.Lines...)
	}

	// Add end marker if we had explicit markers
	if hasExplicitMarkers {
		lines = append(lines, h.generateMarker(BlockEnd))
	}

	// Add trailing lines
	lines = append(lines, config.TrailingLines...)

	// Remove empty trailing element caused by splitting input that ended with \n
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	result := strings.Join(lines, "\n")
	if result != "" {
		result += "\n"
	}
	return []byte(result), nil
}

// generateMarker creates a marker line with the configured comment prefix.
func (h *Handler) generateMarker(blockType BlockType) string {
	prefix := h.CommentPrefix
	if prefix == "" {
		prefix = "#"
	}

	switch blockType {
	case BlockManaged:
		return prefix + " chezmoi:managed"
	case BlockIgnored:
		return prefix + " chezmoi:ignored"
	case BlockEnd:
		return prefix + " chezmoi:end"
	default:
		return ""
	}
}

// GetPath is not supported for plaintext configs.
// Plaintext uses block-based merging instead of path-based access.
func (h *Handler) GetPath(tree any, p path.Path) (any, bool) {
	return nil, false
}

// SetPath is not supported for plaintext configs.
// Plaintext uses block-based merging instead of path-based access.
func (h *Handler) SetPath(tree any, p path.Path, value any) error {
	return fmt.Errorf("SetPath is not supported for plaintext format; use block-based merging")
}

// MergeBlocks performs block-based merging for plaintext configs.
//   - Managed blocks: content from managed (template)
//   - Ignored blocks: content from current config (if available), otherwise from managed
//
// Ignored blocks are matched by index (1st ignored in managed ↔ 1st ignored in current).
func (h *Handler) MergeBlocks(managed, current *ParsedConfig) *ParsedConfig {
	if managed == nil {
		return current
	}

	result := &ParsedConfig{
		CommentPrefix: managed.CommentPrefix,
	}

	// Extract ignored blocks from current config for index-based matching
	currentIgnoredBlocks := extractIgnoredBlocks(current)

	ignoredIndex := 0
	for _, block := range managed.Blocks {
		resultBlock := Block{
			Type:       block.Type,
			MarkerLine: block.MarkerLine,
		}

		if block.Type == BlockManaged {
			// Managed blocks always use template content
			resultBlock.Lines = block.Lines
		} else {
			// Ignored blocks: use current content if available, otherwise template defaults
			if ignoredIndex < len(currentIgnoredBlocks) {
				resultBlock.Lines = currentIgnoredBlocks[ignoredIndex].Lines
				ignoredIndex++
			} else {
				resultBlock.Lines = block.Lines
			}
		}

		result.Blocks = append(result.Blocks, resultBlock)
	}

	return result
}

// extractIgnoredBlocks returns the ignored blocks from current config.
// If current has no markers (all implicit), all content is combined into one block.
func extractIgnoredBlocks(current *ParsedConfig) []Block {
	if current == nil || len(current.Blocks) == 0 {
		return nil
	}

	// If all blocks are implicit (no markers), treat entire content as one ignored block
	if allBlocksImplicit(current) {
		var allLines []string
		for _, block := range current.Blocks {
			allLines = append(allLines, block.Lines...)
		}
		return []Block{{Type: BlockIgnored, Lines: allLines}}
	}

	// Otherwise, collect only the explicitly ignored blocks
	var ignored []Block
	for _, block := range current.Blocks {
		if block.Type == BlockIgnored {
			ignored = append(ignored, block)
		}
	}
	return ignored
}

// allBlocksImplicit returns true if all blocks have no marker lines (implicit blocks).
func allBlocksImplicit(config *ParsedConfig) bool {
	for _, block := range config.Blocks {
		if block.MarkerLine != "" {
			return false
		}
	}
	return true
}

// Ensure Handler implements format.Handler.
var _ format.Handler = (*Handler)(nil)
