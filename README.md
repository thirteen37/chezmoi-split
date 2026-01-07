# chezmoi-split

A [chezmoi](https://chezmoi.io) plugin for managing configuration files that are co-managed by both chezmoi and an application.

## Problem

Some applications (like Zed, VS Code, etc.) modify their configuration files at runtime. When using chezmoi to manage these files, you face a dilemma:

- If chezmoi fully controls the file, the app's runtime changes are lost
- If the app fully controls the file, you can't manage it with chezmoi

## Solution

`chezmoi-split` solves this by:

1. Acting as a script interpreter for chezmoi modify files
2. Merging chezmoi-managed config with app-owned paths from the current file
3. Preserving app's runtime changes while still managing the base configuration

## Installation

```bash
go install github.com/thirteen37/chezmoi-split/cmd/chezmoi-split@latest
```

Make sure `$GOPATH/bin` (usually `~/go/bin`) is in your `PATH` for the plugin to work.

## Usage

### Initialize a split configuration

```bash
chezmoi split init \
  --template zed-settings.json.tmpl \
  --target .config/zed/settings.json \
  --paths '["agent","default_model"]' \
  --paths '["features","edit_prediction_provider"]' \
  --strip-comments

# Or from an existing config file:
chezmoi split init \
  --from ~/.config/zed/settings.json \
  --target .config/zed/settings.json \
  --paths '["agent","default_model"]'
```

This creates a single modify script in your chezmoi source directory.

### Generated file format

```
#!/usr/bin/env chezmoi-split
version 1

format json
strip-comments true

ignore ["agent", "default_model"]
ignore ["features", "edit_prediction_provider"]
ignore ["context_servers", "*", "enabled"]

chezmoi:modify-template
{
  "base_keymap": "VSCode",
  "vim_mode": true,
  "context_servers": {
    "mcp-server-github": {
      "settings": {
        "github_personal_access_token": "{{ onepasswordRead "op://Vault/Item/credential" }}"
      }
    }
  },
  "agent": {
    "default_model": {
      "provider": "zed.dev",
      "model": "claude-sonnet-4"
    }
  }
}
```

### How it works

1. **Chezmoi** sees the `.tmpl` suffix and renders all template syntax first
   - `{{ onepasswordRead "..." }}` becomes the actual secret
   - `{{ .chezmoi.homeDir }}` becomes `/Users/you`
2. **Chezmoi** sees `chezmoi:modify-template` and removes that line
3. **Chezmoi** executes the script via shebang (`chezmoi-split`)
4. **chezmoi-split** parses directives and managed config, reads current file from stdin
5. **chezmoi-split** merges them, preserving `ignore` paths from current, outputs result

### Directives

| Directive | Description | Example |
|-----------|-------------|---------|
| `version` | Format version (required, must be first) | `version 1` |
| `format` | Config format (default: auto-detect) | `format json` |
| `strip-comments` | Strip // comments from JSON | `strip-comments true` |
| `ignore` | Path to preserve from current file | `ignore ["agent", "model"]` |

### Example

**Managed config (in script):**
```json
{
  "base_keymap": "VSCode",
  "agent": {
    "default_model": {"provider": "default", "model": "default-model"},
    "profiles": {"ask": {"tools": ["read_file"]}}
  }
}
```

**Current file (with app's runtime changes):**
```json
{
  "base_keymap": "VSCode",
  "agent": {
    "default_model": {"provider": "user-choice", "model": "claude-sonnet"},
    "profiles": {"ask": {"tools": ["read_file"]}}
  }
}
```

**Ignore paths:** `["agent", "default_model"]`

**Result after merge:**
```json
{
  "base_keymap": "VSCode",
  "agent": {
    "default_model": {"provider": "user-choice", "model": "claude-sonnet"},
    "profiles": {"ask": {"tools": ["read_file"]}}
  }
}
```

The `agent.default_model` is preserved from current because it's ignored, while the rest comes from the managed config.

## Features

- **Single file**: Directives and template in one modify script
- **Chezmoi templating**: Full support for secrets, variables, conditionals
- **JSON/JSONC support**: Can strip `//` comments from JSON files
- **Nested paths**: Supports deep path selectors like `["context_servers", "mcp-server", "enabled"]`
- **Versioned format**: Built-in versioning for future migrations

## License

MIT
