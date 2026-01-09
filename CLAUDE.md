# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

**Keep documentation up to date.** When adding new features, formats, directives, or changing behavior, update both this file and README.md.

## Build & Test Commands

```bash
go build ./...                          # Build all packages
go test ./...                           # Run all tests
go test -v -race -coverprofile=coverage.out ./...  # Run tests with race detection and coverage
go test ./internal/merge/...            # Run tests for a specific package
golangci-lint run                       # Lint (used in CI)
go install ./cmd/chezmoi-split          # Install locally
```

## Architecture

chezmoi-split is a script interpreter for chezmoi modify scripts. It manages configuration files that are co-managed by both chezmoi and an application (like Zed, VS Code).

When invoked via shebang (`#!/usr/bin/env chezmoi-split`), it reads the script file, parses directives, reads current config from stdin, and outputs merged config.

### Core Packages

- **`internal/script`**: Parses the script format (version, format, strip-comments, ignore directives, header, and template content)
- **`internal/merge`**: Core merge algorithm - starts with managed config, overlays values from current config at ignored paths
- **`internal/format`**: Handler interface for config formats (Parse, Serialize, GetPath, SetPath)
- **`internal/format/json`**: JSON/JSONC handler with wildcard path support
- **`internal/format/toml`**: TOML handler with full nested path support
- **`internal/format/ini`**: INI handler (section.key paths only, all values as strings)
- **`internal/format/plaintext`**: Plaintext handler with block-based merging using markers (`chezmoi:managed`, `chezmoi:ignored`, `chezmoi:end`)
- **`internal/path`**: Path selector abstraction for navigating config trees (e.g., `["agent", "default_model"]`)

### Script Format

Scripts combine directives and template in one file:
```
#!/usr/bin/env chezmoi-split
# version 1
# format json
# strip-comments true
# ignore ["path", "to", "ignore"]
#---
{ "config": "here" }
```

Directives are prefixed with `#` and the `#---` separator marks the start of the template content. Shebang lines (`#!`) are automatically skipped.

**Directive rules:**
- `version` is required and must be the first directive
- `format` defaults to `auto` (uses JSON handler) if not specified
- `ignore` and `strip-comments` emit warnings when used with plaintext format (they don't apply)

Supported formats: `json`, `toml`, `ini`, `plaintext`, `auto` (auto-detect)

For plaintext format, markers (`chezmoi:managed`, `chezmoi:ignored`, `chezmoi:end`) are preserved exactly as written in the template. You can format them however you want: `# chezmoi:managed`, `// chezmoi:managed`, `" chezmoi:managed`, etc.

### Format Handler Details

**JSON/JSONC:**
- Preserves key order using ordered maps
- Wildcard paths (`*`) supported at any level
- `strip-comments` removes single-line `//` comments

**TOML:**
- Preserves key order using ordered maps
- Wildcard paths supported
- `strip-comments` not supported (returns error)

**INI:**
- Path depth limited to 2 segments: `["section"]` or `["section", "key"]`
- All values stored as strings
- Global keys stored under empty string key (`""`)
- `strip-comments` not supported (returns error)

**Plaintext:**
- Marker detection is substring-based (no escape mechanism)
- Content before any marker is treated as an implicit ignored block
- Index-based matching: 1st ignored block in template matches 1st ignored block in current

### Merge Algorithm

**Structured formats (JSON, TOML, INI):**
1. Deep copy managed config as base (preserves ordered maps and slices)
2. For each ignored path, if it exists in current config, overlay that value onto result
3. If ignored path doesn't exist in current config, keeps the managed value (not deleted)
4. This preserves app-managed values while applying chezmoi-managed structure

**Plaintext format:**
1. Uses block-based merging with markers (`chezmoi:managed`, `chezmoi:ignored`, `chezmoi:end`)
2. Managed blocks: content always from template
3. Ignored blocks: content from current config (matched by index), falls back to template defaults
4. If current config has no markers, all content is treated as one implicit ignored block
