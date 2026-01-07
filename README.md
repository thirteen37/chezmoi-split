# chezmoi-split

A [chezmoi](https://chezmoi.io) plugin for managing configuration files that are co-managed by both chezmoi and an application.

## Problem

Some applications (like Zed, VS Code, etc.) modify their configuration files at runtime. When using chezmoi to manage these files, you face a dilemma:

- If chezmoi fully controls the file, the app's runtime changes are lost
- If the app fully controls the file, you can't manage it with chezmoi

## Solution

`chezmoi-split` solves this by:

1. Allowing you to define which parts of the config are "app-owned"
2. Generating modify scripts that merge chezmoi-managed config with app-owned paths
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
```

This creates:
- A modify script in your chezmoi source directory
- A paths configuration file listing app-owned paths

### Add app-owned paths

```bash
chezmoi split add-path .config/zed/settings.json '["context_servers","OpenDia","enabled"]'
```

### Remove app-owned paths

```bash
chezmoi split remove-path .config/zed/settings.json '["agent","default_model"]'
```

### List app-owned paths

```bash
chezmoi split list .config/zed/settings.json
```

## How it works

1. You create a chezmoi template with your managed configuration
2. `chezmoi split init` generates a modify script that:
   - Renders your template to get the managed config
   - Reads the current file (with app's changes)
   - Merges them, preserving app-owned paths from current
3. When you run `chezmoi apply`, the modify script runs and produces the merged result

### Example

**Managed config (template):**
```json
{
  "base_keymap": "VSCode",
  "agent": {
    "default_model": {"provider": "default", "model": "default-model"},
    "profiles": {"ask": {"tools": ["read_file"]}}
  }
}
```

**Current file (with app changes):**
```json
{
  "base_keymap": "VSCode",
  "agent": {
    "default_model": {"provider": "user-choice", "model": "claude-sonnet"},
    "profiles": {"ask": {"tools": ["read_file"]}}
  }
}
```

**App-owned paths:** `["agent", "default_model"]`

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

The `agent.default_model` is preserved from current because it's app-owned, while the rest comes from the managed template.

## Features

- **JSON/JSONC support**: Can strip `//` comments from JSON files
- **Nested paths**: Supports deep path selectors like `["context_servers", "mcp-server", "enabled"]`
- **Extensible architecture**: Designed to support YAML, TOML, INI in future versions

## License

MIT
