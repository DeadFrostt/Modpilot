# Modpilot

**Modpilot** is a lightweight Modrinth modpack manager written in Go. Define modpacks as collections of Modrinth slugs in a JSON config, then add, remove, update, and sync mods via a simple CLI.

## Features

- Manage multiple modpacks in a single `config.json`
+- Define specific Minecraft versions and loaders for each modpack
- Bulk add/remove of Modrinth slugs
- Interactive `init` for setting *optional* global defaults (MC version & loader)
- Precise version filtering (game versions & loader compatibility)
- Sync local mods folder to config
- Verbose logging and automatic confirmation via `--yes`
- Command aliases for faster workflows

## Prerequisites

- Go 1.18 or later

## Installation

Clone the repo and build:
```pwsh
git clone https://github.com/yourusername/modpilot.git
cd modpilot
go build -o modpilot.exe main.go
```

Or download a prebuilt binary for your platform.

## Quick Start

1. Initialize your workspace (sets optional global defaults):
   ```pwsh
   .\modpilot.exe init -c config.json -s state.json
   ```
2. Create a new modpack (prompts for MC version & loader):
   ```pwsh
   .\modpilot.exe create-pack MyPack
   ```
3. Add mods to the pack:
   ```pwsh
   .\modpilot.exe add-mod MyPack mod-slug-one mod-slug-two
   ```
4. List modpacks and their mods:
   ```pwsh
   .\modpilot.exe list-packs
   .\modpilot.exe list-mods MyPack
   ```
5. Check for updates (uses pack's configured version/loader):
   ```pwsh
   .\modpilot.exe update MyPack --yes --verbose
   # Optionally override version/loader for this run:
   .\modpilot.exe update MyPack -g 1.20.1 -l forge
   ```
6. View up-to-date/outdated status (uses pack's configured version/loader):
   ```pwsh
   .\modpilot.exe status MyPack
   ```
7. Remove jars not listed in config:
   ```pwsh
   .\modpilot.exe sync MyPack --verbose
   # or use alias:
   .\modpilot.exe clean MyPack
   ```

## Commands

| Command                   | Alias(es)             | Description                                             |
|---------------------------|-----------------------|---------------------------------------------------------|
| init                      |                       | Initialize config, setting optional global defaults     |
| create-pack [name]        |                       | Create a new modpack, prompting for its settings        |
| delete-pack [name]        |                       | Delete a modpack from config                            |
| list-packs                | lp                    | List all modpacks and their settings                    |
| list-mods [pack]          | lm                    | List all mods in a modpack                              |
| add-mod [pack] [slugs...] |                       | Add one or more Modrinth slugs to a modpack             |
| remove-mod [pack] [slugs...] |                    | Remove one or more slugs from a modpack                 |
-| set-mc [version]          |                       | Set default Minecraft version in `config.json`          |
-| set-loader [loader]       |                       | Set default mod loader (fabric/forge/…) in `config.json` |
| update [pack]             | update-pack, upd      | Check & download new versions for a modpack             |
| status [pack]             |                       | Display up-to-date/outdated status for a modpack        |
| sync [pack]               | sync-pack, clean      | Remove JARs not listed in the modpack config            |

Global flags: `-c, --config`, `-s, --state`, `-m, --mods-dir`, `-y, --yes`, `-g, --mc-version` (override), `-l, --loader` (override), `-v, --verbose`.

## Configuration (`config.json`)

```json
{
  "default_mc_version": "1.21.5",
  "default_loader": "fabric",
  "modpacks": {
    "MyPack": {
      "mc_version": "1.21.5",
      "loader": "fabric",
      "mods": [
        "mod-slug-one",
        "mod-slug-two"
      ]
    },
    "AnotherPack": {
      "mc_version": "1.20.5",
      "loader": "fabric",
      "mods": []
    }
  }
}
```

- `default_mc_version` (optional): Suggested MC version when creating new packs.
- `default_loader` (optional): Suggested loader when creating new packs.
- `modpacks`: Map where each key is a pack name.
  - `mc_version`: Minecraft version specific to this pack.
  - `loader`: Mod loader specific to this pack.
  - `mods`: Array of Modrinth slugs for this pack.

## State (`state.json`)

Keeps track of last-downloaded version IDs per pack and slug:

```json
{
  "MyPack": {
    "mod-slug-one": "version-id",
    "mod-slug-two": "version-id"
  }
}
```

## Mods Directory

By default, JARs are downloaded to `<mods-dir>/<pack>/`. Override with `--mods-dir`.

## Aliases

- `modpilot` (alias `modpm`, `mp`)
- `list-packs` (`lp`)
- `list-mods` (`lm`)
- `update` (`update-pack`, `upd`)
- `sync` (`sync-pack`, `clean`)

## Quality of Life Improvements

- Command aliases for faster typing
+- Per-pack Minecraft version and loader configuration
- Automatic filtering by `game_versions` and `loaders` to avoid incompatible jars
- Consistent download path structure

## Contributing

Contributions welcome! Open issues or submit pull requests on GitHub.
