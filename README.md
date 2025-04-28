# Modpilot

**Modpilot** is a lightweight Modrinth modpack manager written in Go. Define modpacks as collections of Modrinth slugs in a JSON config, then add, remove, check for updates, update, and sync mods via a simple CLI.

## Features

- Manage multiple modpacks in a single `config.json`
- Define specific Minecraft versions and loaders for each modpack (**required**)
- Bulk add/remove of Modrinth slugs
- Interactive `init` for setting *optional* global defaults (MC version & loader)
- Precise version filtering (game versions & loader compatibility)
- Checks for updates against Modrinth (`check-updates`)
- Downloads/updates mods, storing version ID and filename in `state.json` (`update`)
- Checks for missing local mod files during updates and checks
- Syncs local mods folder based on `state.json`, removing unexpected JARs (`sync`)
- Configuration validation on load
- Verbose logging and automatic confirmation via `--yes`
- Command aliases for faster workflows

## Prerequisites

- Go 1.18 or later

## Installation

Clone the repo and build:
```pwsh
git clone https://github.com/yourusername/modpilot.git # Replace with actual repo URL if available
cd modpilot
go build -o modpilot.exe .
```

Or download a prebuilt binary for your platform if available.

## Quick Start

1.  Initialize your workspace (sets optional global defaults):
    ```pwsh
    .\modpilot.exe init
    ```
2.  Create a new modpack (prompts for required MC version & loader):
    ```pwsh
    .\modpilot.exe create-pack MyPack
    ```
3.  Add mods to the pack:
    ```pwsh
    .\modpilot.exe add-mod MyPack fabric-api sodium iris
    ```
4.  List modpacks and their mods:
    ```pwsh
    .\modpilot.exe list-packs
    .\modpilot.exe list-mods MyPack
    ```
5.  Check for available updates and missing files:
    ```pwsh
    .\modpilot.exe check-updates MyPack
    ```
6.  Download/update mods (uses pack's configured version/loader):
    ```pwsh
    .\modpilot.exe update MyPack --yes --verbose
    # Optionally override version/loader for this run:
    # .\modpilot.exe update MyPack -g 1.20.1 -l forge
    ```
7.  Remove mods (from config and state):
    ```pwsh
    .\modpilot.exe remove-mod MyPack sodium
    ```
8.  Clean up unexpected JARs from the mods folder (based on `state.json`):
    ```pwsh
    .\modpilot.exe sync MyPack --verbose
    # or use alias:
    # .\modpilot.exe clean MyPack
    ```

## Commands

| Command                      | Alias(es)        | Description                                                                 |
|------------------------------|------------------|-----------------------------------------------------------------------------|
| `init`                       |                  | Initialize config, setting optional global defaults                         |
| `create-pack [name]`         |                  | Create a new modpack, prompting for its required settings                   |
| `delete-pack [name]`         |                  | Delete a modpack from config (doesn't delete state or files yet)            |
| `list-packs`                 | `lp`             | List all modpacks and their settings                                        |
| `list-mods [pack]`           | `lm`             | List all mods configured for a modpack                                      |
| `add-mod [pack] [slugs...]`  |                  | Add one or more Modrinth slugs to a modpack's config                        |
| `remove-mod [pack] [slugs...]`|                 | Remove one or more slugs from a modpack's config and state                  |
| `check-updates [pack]`       |                  | Check Modrinth for newer versions and check for missing local files         |
| `update [pack]`              | `update-pack`, `upd` | Check & download new/missing versions for a modpack, updating state         |
| `sync [pack]`                | `sync-pack`, `clean` | Remove JARs from the modpack's directory that aren't listed in `state.json` |

Global flags: `-c, --config`, `-s, --state`, `-m, --mods-dir`, `-y, --yes`, `-g, --mc-version` (override), `-l, --loader` (override), `-v, --verbose`.

## Configuration (`config.json`)

```json
{
  "default_mc_version": "1.21.5", // Optional: Used as default during 'create-pack'
  "default_loader": "fabric",   // Optional: Used as default during 'create-pack'
  "modpacks": {
    "MyPack": {
      "mc_version": "1.21.5", // Required: Minecraft version for this pack
      "loader": "fabric",   // Required: Mod loader for this pack
      "mods": [             // Required: Array of Modrinth slugs
        "fabric-api",
        "sodium"
      ]
    },
    "AnotherPack": {
      "mc_version": "1.20.1",
      "loader": "forge",
      "mods": []
    }
    // ... more packs
  }
}
```

- `default_mc_version` (optional): Suggested MC version when creating new packs.
- `default_loader` (optional): Suggested loader when creating new packs.
- `modpacks`: Map where each key is a pack name.
  - `mc_version` (**Required**): Minecraft version specific to this pack.
  - `loader` (**Required**): Mod loader specific to this pack (e.g., "fabric", "forge", "quilt", "neoforge").
  - `mods` (**Required**): Array of Modrinth slugs for this pack.

*Validation*: The tool checks that `mc_version` and `loader` are present for each pack when loading the config.

## State (`state.json`)

Keeps track of the last-downloaded version ID and filename for each mod in a pack. This file is used by `update`, `check-updates`, and `sync`.

```json
{
  "MyPack": {
    "fabric-api": {
      "version_id": "abcdef12",
      "filename": "fabric-api-0.100.0+1.21.5.jar"
    },
    "sodium": {
      "version_id": "xyz789uv",
      "filename": "sodium-fabric-mc1.21.5-0.5.9.jar"
    }
  },
  "AnotherPack": {
    // ... mods for this pack
  }
}
```

- Each key under the pack name is the mod slug.
- `version_id`: The Modrinth version ID that was last downloaded/checked.
- `filename`: The actual filename of the JAR file that was downloaded for that version.

## Mods Directory

By default, JARs are downloaded to `<mods-dir>/<packName>/`. The default `<mods-dir>` is `mods` in the current directory. Override the base directory with the `--mods-dir` flag.

Example: `mods/MyPack/fabric-api-0.100.0+1.21.5.jar`

## Aliases

- `modpilot` (alias `modpm`, `mp`)
- `list-packs` (`lp`)
- `list-mods` (`lm`)
- `update` (`update-pack`, `upd`)
- `sync` (`sync-pack`, `clean`)
// Note: check-updates does not have an alias currently

## Contributing

Contributions welcome! Open issues or submit pull requests on GitHub.
