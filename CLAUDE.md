# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

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

- **`internal/script`**: Parses the script format (version, format, strip-comments, comment-prefix, ignore directives, header, and template content)
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

Directives are prefixed with `#` and the `#---` separator marks the start of the template content.

Supported formats: `json`, `toml`, `ini`, `plaintext`, `auto` (auto-detect)

For plaintext, use `comment-prefix` directive (presets: `shell`, `vim`, `c`, `lua`, `sql`, `semicolon` or literal value).

### Merge Algorithm

**Structured formats (JSON, TOML, INI):**
1. Deep copy managed config as base
2. For each ignored path, if it exists in current config, overlay that value onto result
3. This preserves app-managed values while applying chezmoi-managed structure

**Plaintext format:**
1. Uses block-based merging with markers (`chezmoi:managed`, `chezmoi:ignored`, `chezmoi:end`)
2. Managed blocks: content always from template
3. Ignored blocks: content from current config (matched by index), falls back to template defaults
