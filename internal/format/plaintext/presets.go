package plaintext

// CommentPresets maps preset names to their actual comment prefix strings.
var CommentPresets = map[string]string{
	"shell":     "#",  // bash, zsh, sh, tmux.conf, .gitconfig
	"vim":       "\"", // .vimrc
	"c":         "//", // C-style comments
	"semicolon": ";",  // some INI-style configs
	"lua":       "--", // lua configs (e.g., neovim)
	"sql":       "--", // SQL-style comments
}

// ResolveCommentPrefix resolves a comment prefix value.
// If the value is a known preset name, returns the preset's prefix.
// Otherwise, returns the value as-is (treating it as a literal prefix).
// Surrounding quotes are stripped from literal values.
func ResolveCommentPrefix(value string) string {
	// Check if it's a preset name
	if prefix, ok := CommentPresets[value]; ok {
		return prefix
	}

	// Treat as literal - strip surrounding quotes if present
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}

	return value
}
