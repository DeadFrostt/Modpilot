# modpilot

**modpilot** is a lightweight Modrinth modpack manager written in Go. It allows you to define modpacks as collections of Modrinth slugs in a JSON config, then add, remove, update, and sync mods via a simple CLI.

## Features

- Define multiple modpacks in `config.json`
- Add or remove one or many mods at once
- Interactive `init` command to set up defaults (MC version, loader)
- Bulk operations with `yes` and `verbose` flags
- Subcommands for:
  - Creating/deleting modpacks
  - Listing modpacks and their mods
  - Setting default Minecraft version and loader
  - Checking for and downloading updates
  - Showing status (up-to-date/outdated)
  - Syncing (removing unlisted jars)

## Prerequisites

- Go 1.18+

## Installation

Clone the repo and build:

```pwsh
git clone https://github.com/yourusername/modpilot.git
cd modpilot
go build -o modpilot.exe main.go
```

Or download the latest release for Windows.

## Quick Start

1. Initialize your workspace:
   ```pwsh
   .\modpilot.exe init -c config.json -s state.json
   ```
   This prompts for default Minecraft version and loader, and creates `config.json` and `state.json` if missing.

2. Create a new modpack:
   ```pwsh
   .\modpilot.exe create-pack TechPack
   ```

3. Add mods:
   ```pwsh
   .\modpilot.exe add-mod TechPack ae2 thermal-foundation
   ```

4. List modpacks and mods:
   ```pwsh
   .\modpilot.exe list-packs
   .\modpilot.exe list-mods TechPack
   ```

5. Check for updates and download:
   ```pwsh
   .\modpilot.exe update TechPack --yes --verbose
   ```

6. Show status without downloading:
   ```pwsh
   .\modpilot.exe status TechPack
   ```

7. Sync local mods folder (remove extra jars):
   ```pwsh
   .\modpilot.exe sync TechPack --verbose
   ```

## Commands

| Command                   | Description                                      |
|---------------------------|--------------------------------------------------|
| init                      | Initialize or update config/state interactively  |
| create-pack [name]        | Create a new modpack                            |
| delete-pack [name]        | Remove a modpack                                |
| list-packs (lp)           | List all modpack names                           |
| list-mods [pack] (lm)     | List all mods in a modpack                       |
| add-mod [pack] [slugs...] | Add one or more Modrinth slugs to a modpack      |
| remove-mod [pack] [slugs...] | Remove one or more slugs from a modpack        |
| set-mc [version]          | Set default Minecraft version in `config.json`   |
| set-loader [loader]       | Set default mod loader (fabric/forge/...)        |
| update [pack] (upd)       | Check & download new versions for a modpack      |
| status [pack]             | Display status of mods (up-to-date/outdated)     |
| sync [pack] (clean)       | Remove JARs not listed in the modpack config     |

Use `-c`, `-s`, `-m`, `-y`, `-g`, `-l`, and `-v` flags for config path, state path, mods directory, yes (auto-confirm), MC version override, loader override, and verbose output, respectively.

## Configuration

`config.json` format:

```json
{
  "mc_version": "1.19.2",
  "loader": "fabric",
  "modpacks": {
    "TechPack": ["ae2", "thermal-foundation"]
  }
}
```

- `mc_version`: Default Minecraft version
- `loader`: Default mod loader
- `modpacks`: Map of pack names to arrays of Modrinth slugs

## State File

`state.json` keeps track of the last-downloaded version IDs per pack and slug:

```json
{
  "TechPack": {
    "ae2": "version-id",
    "thermal-foundation": "version-id"
  }
}
```

## Mods Directory

By default, downloaded JARs go into `mods/<pack>/`. Override with `--mods-dir`.

## Contributing

Contributions welcome! Feel free to open issues or submit pull requests.

## License

MIT © Your Name

---

Updated 2025-04-27

## Quality of Life Improvements

- Aliases for most commands to speed up typing
- Precise version filtering: now the tool only downloads mods whose `game_versions` and `loaders` lists match your chosen Minecraft version/loader. Falls back to the closest match or errors if none found.
- Correct destDir usage for downloads so files are always placed under `<mods-dir>/<pack>/`.

### Aliases

- `modpilot` (alias `modpm`, `mp`)
- `update` (alias `update-pack`, `upd`)
- `list-packs` (alias `lp`)
- `list-mods` (alias `lm`)
- `sync` (alias `sync-pack`, `clean`)
