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

Create a modify script in your chezmoi source directory. For example, to manage `~/.config/zed/settings.json`:

```bash
mkdir -p ~/.local/share/chezmoi/dot_config/zed
touch ~/.local/share/chezmoi/dot_config/zed/modify_settings.json.tmpl
chmod +x ~/.local/share/chezmoi/dot_config/zed/modify_settings.json.tmpl
```

### Script format

```
#!/usr/bin/env chezmoi-split
# version 1
# format json
# strip-comments true
# ignore ["agent", "default_model"]
# ignore ["features", "edit_prediction_provider"]
# ignore ["context_servers", "*", "enabled"]
#---
// My comments for the final JSON file
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
2. **Chezmoi** executes the modify script via shebang (`chezmoi-split`)
3. **chezmoi-split** parses directives (lines starting with `#`) until `#---` separator
4. **chezmoi-split** reads managed config from template section, current file from stdin
5. **chezmoi-split** merges them, preserving `ignore` paths from current, outputs result

### Directives

| Directive | Description | Example |
|-----------|-------------|---------|
| `version` | Format version (required, must be first) | `# version 1` |
| `format` | Config format: `json`, `toml`, or `ini` | `# format json` |
| `strip-comments` | Strip `//` comments from JSON before parsing | `# strip-comments true` |
| `ignore` | Path to preserve from current file | `# ignore ["agent", "model"]` |

The `#---` line marks the boundary between directives and template content. Lines before the JSON (like `// comments`) are preserved in the output.

### Ignore paths

Ignore paths use JSON array syntax to specify nested keys:

| Path | Matches |
|------|---------|
| `["agent"]` | The entire `agent` object |
| `["agent", "default_model"]` | Only `agent.default_model` |
| `["servers", "*", "enabled"]` | `enabled` field in ALL objects under `servers` |

**Wildcard (`*`)**: Matches any key at that level. Useful for preserving a field across all items in an object.

**Format-specific notes:**
- **JSON/TOML**: Full nested path support (any depth)
- **INI**: Paths limited to `["section", "key"]` (2 levels max)

### Merge behavior

- **Ignored path exists in current**: Value from current file is used
- **Ignored path missing in current**: Value from managed config is used (not deleted)
- **Path not ignored**: Value from managed config always wins

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

### TOML example

```
#!/usr/bin/env chezmoi-split
# version 1
# format toml
# ignore ["user", "preferences"]
#---
[server]
host = "localhost"
port = 8080

[user]
name = "default"
preferences = { theme = "dark" }
```

TOML supports full nested paths like JSON (e.g., `["server", "tls", "enabled"]`).

### INI example

```
#!/usr/bin/env chezmoi-split
# version 1
# format ini
# ignore ["database", "password"]
#---
[database]
host = localhost
port = 3306
password = default

[server]
address = 0.0.0.0
```

INI paths are limited to section and key: `["section", "key"]`.

## Features

- **Single file**: Directives and template in one modify script
- **Chezmoi templating**: Full support for secrets, variables, conditionals
- **Multiple formats**: JSON, TOML, and INI support
- **JSON/JSONC support**: Can strip `//` comments from JSON files
- **Header preservation**: Comments before the config are passed through to output
- **Wildcard paths**: Use `*` to match any key at a path level
- **Versioned format**: Built-in versioning for future migrations

## License

MIT
