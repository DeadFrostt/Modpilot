package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	defaultConfig = "config.json"
	defaultState  = "state.json"
	defaultMods   = "mods"
)

var (
	cfgFile       string
	stateFile     string
	modsDir       string
	autoYes       bool
	mcVersionFlag string // override MC version
	loaderFlag    string // override loader
	verbose       bool // enable verbose logging
)

func main() {
	root := &cobra.Command{
		Use:     "modpilot",
		Aliases: []string{"modpm", "mp"},
		Short:   "modpilot — a Modrinth modpack manager",
		Long:    "Define modpack “stacks” in config.json, then list, add, remove, or update mods via the CLI.",
	}

	// Global flags
	root.PersistentFlags().StringVarP(&cfgFile, "config", "c", defaultConfig, "path to config.json")
	root.PersistentFlags().StringVarP(&stateFile, "state", "s", defaultState, "path to state.json")
	root.PersistentFlags().StringVarP(&modsDir, "mods-dir", "m", defaultMods, "where to drop downloaded JARs")
	root.PersistentFlags().BoolVarP(&autoYes, "yes", "y", false, "auto-confirm updates")
	root.PersistentFlags().StringVarP(&mcVersionFlag, "mc-version", "g", "", "override Minecraft version (e.g. 1.18.2)")
	root.PersistentFlags().StringVarP(&loaderFlag, "loader", "l", "", "override mod loader (fabric|forge|…)")
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")

	// list-packs
	listPacks := &cobra.Command{
		Use:     "list-packs",
		Aliases: []string{"lp"},
		Short:   "List all defined modpacks and their settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := LoadConfig(cfgFile)
			if err != nil {
				return err
			}
			fmt.Println("Modpacks:")
			for name, packCfg := range cfg.Modpacks {
				fmt.Printf(" • %s (MC: %s, Loader: %s)\n", name, packCfg.MCVersion, packCfg.Loader)
			}
			return nil
		},
	}

	// list-mods
	listMods := &cobra.Command{
		Use:     "list-mods [modpack]",
		Aliases: []string{"lm"},
		Short:   "List all mods in a modpack",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packName := args[0]
			cfg, err := LoadConfig(cfgFile)
			if err != nil {
				return err
			}
			packCfg, ok := cfg.Modpacks[packName]
			if !ok {
				return fmt.Errorf("modpack %q not found", packName)
			}
			fmt.Printf("Mods in %s (MC: %s, Loader: %s):\n", packName, packCfg.MCVersion, packCfg.Loader)
			for _, slug := range packCfg.Mods {
				fmt.Printf(" • %s\n", slug)
			}
			return nil
		},
	}

	// add-mod
	addMod := &cobra.Command{
		Use:   "add-mod [modpack] [modSlugs...]",
		Short: "Add one or more Modrinth slugs to a modpack",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			packName := args[0]
			slugs := args[1:]
			cfg, err := LoadConfig(cfgFile)
			if err != nil {
				return err
			}
			packCfg, ok := cfg.Modpacks[packName]
			if !ok {
				return fmt.Errorf("modpack %q not found", packName)
			}
			changed := false
			for _, slug := range slugs {
				exists := false
				for _, m := range packCfg.Mods {
					if m == slug {
						fmt.Printf("%q already in %s\n", slug, packName)
						exists = true
						break
					}
				}
				if !exists {
					packCfg.Mods = append(packCfg.Mods, slug)
					fmt.Printf("Added %q to %s\n", slug, packName)
					changed = true
				}
			}
			if changed {
				cfg.Modpacks[packName] = packCfg // Update the map entry
				if err := SaveConfig(cfgFile, cfg); err != nil {
					return err
				}
			}
			return nil
		},
	}

	// remove-mod
	removeMod := &cobra.Command{
		Use:   "remove-mod [modpack] [modSlugs...]",
		Short: "Remove one or more Modrinth slugs from a modpack",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			packName := args[0]
			rem := args[1:]
			cfg, err := LoadConfig(cfgFile)
			if err != nil {
				return err
			}
			packCfg, ok := cfg.Modpacks[packName]
			if !ok {
				return fmt.Errorf("modpack %q not found", packName)
			}
			origLen := len(packCfg.Mods)
			for _, slug := range rem {
				found := false
				newList := make([]string, 0, len(packCfg.Mods))
				for _, m := range packCfg.Mods {
					if m == slug {
						found = true
					} else {
						newList = append(newList, m)
					}
				}
				if !found {
					fmt.Printf("%q not in %s\n", slug, packName)
				} else {
					packCfg.Mods = newList
					fmt.Printf("Removed %q from %s\n", slug, packName)
				}
			}
			if len(packCfg.Mods) != origLen {
				cfg.Modpacks[packName] = packCfg // Update the map entry
				if err := SaveConfig(cfgFile, cfg); err != nil {
					return err
				}

				// Also remove from state
				state, err := LoadState(stateFile)
				if err != nil {
					// Warn but don't fail the whole operation if state load fails
					fmt.Printf("Warning: could not load state file to remove mod entries: %v\n", err)
				} else if packState, ok := state[packName]; ok {
					stateChanged := false
					for _, slug := range rem {
						if _, exists := packState[slug]; exists {
							delete(packState, slug)
							stateChanged = true
							if verbose {
								fmt.Printf("Removed %q from state for %s\n", slug, packName)
							}
						}
					}
					if stateChanged {
						if err := SaveState(stateFile, state); err != nil {
							fmt.Printf("Warning: could not save updated state file: %v\n", err)
						}
					}
				}
			}
			return nil
		},
	}

	// create-pack
	createPack := &cobra.Command{
		Use:   "create-pack [modpack]",
		Short: "Create a new modpack in the config, prompting for settings",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfg, err := LoadConfig(cfgFile)
			if err != nil && !os.IsNotExist(err) {
				return err
			} else if err != nil { // File doesn't exist, create new config
				cfg = &Config{Modpacks: make(map[string]ModpackConfig)}
			}
			if _, exists := cfg.Modpacks[name]; exists {
				fmt.Printf("modpack %q already exists\n", name)
				return nil
			}

			reader := bufio.NewReader(os.Stdin)
			var mcVersion, loader string

			// Prompt for MC Version, using default if available
			promptMC := fmt.Sprintf("Enter Minecraft version for %s", name)
			if cfg.DefaultMCVersion != "" {
				promptMC += fmt.Sprintf(" (default: %s)", cfg.DefaultMCVersion)
			}
			fmt.Print(promptMC + ": ")
			mcVersion, _ = reader.ReadString('\n')
			mcVersion = strings.TrimSpace(mcVersion)
			if mcVersion == "" {
				mcVersion = cfg.DefaultMCVersion // Use default if input is empty
			}
			if mcVersion == "" {
				return fmt.Errorf("Minecraft version cannot be empty")
			}

			// Prompt for Loader, using default if available
			promptLoader := fmt.Sprintf("Enter mod loader for %s (e.g., fabric, forge)", name)
			if cfg.DefaultLoader != "" {
				promptLoader += fmt.Sprintf(" (default: %s)", cfg.DefaultLoader)
			}
			fmt.Print(promptLoader + ": ")
			loader, _ = reader.ReadString('\n')
			loader = strings.TrimSpace(loader)
			if loader == "" {
				loader = cfg.DefaultLoader // Use default if input is empty
			}
			if loader == "" {
				return fmt.Errorf("mod loader cannot be empty")
			}

			cfg.Modpacks[name] = ModpackConfig{
				MCVersion: mcVersion,
				Loader:    loader,
				Mods:      []string{},
			}
			if err := SaveConfig(cfgFile, cfg); err != nil {
				return err
			}
			fmt.Printf("Created modpack %q with MC %s and loader %s\n", name, mcVersion, loader)
			return nil
		},
	}

	// delete-pack
	deletePack := &cobra.Command{
		Use:   "delete-pack [modpack]",
		Short: "Delete a modpack from the config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfg, err := LoadConfig(cfgFile)
			if err != nil {
				return err
			}
			if _, exists := cfg.Modpacks[name]; !exists {
				fmt.Printf("modpack %q not found\n", name)
				return nil
			}
			delete(cfg.Modpacks, name)
			if err := SaveConfig(cfgFile, cfg); err != nil {
				return err
			}
			fmt.Printf("Deleted modpack %q\n", name)
			// Also remove associated state and mods directory?
			// state, _ := LoadState(stateFile)
			// delete(state, name)
			// SaveState(stateFile, state)
			// os.RemoveAll(filepath.Join(modsDir, name))
			return nil
		},
	}

	// REMOVED set-mc and set-loader commands as they are now per-pack

	// init
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize or update config/state files, setting global defaults",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := LoadConfig(cfgFile)
			if err != nil && !os.IsNotExist(err) {
				return err
			} else if err != nil { // File doesn't exist, create new config
				cfg = &Config{Modpacks: make(map[string]ModpackConfig)}
			}
			reader := bufio.NewReader(os.Stdin)

			// Prompt for Default MC Version
			fmt.Printf("Enter default Minecraft version (optional, e.g. 1.19.2) [%s]: ", cfg.DefaultMCVersion)
			v, _ := reader.ReadString('\n')
			v = strings.TrimSpace(v)
			if v != "" { // Only update if user provided input
				cfg.DefaultMCVersion = v
			}

			// Prompt for Default Loader
			fmt.Printf("Enter default mod loader (optional, e.g. fabric/forge) [%s]: ", cfg.DefaultLoader)
			l, _ := reader.ReadString('\n')
			l = strings.TrimSpace(l)
			if l != "" { // Only update if user provided input
				cfg.DefaultLoader = l
			}

			if err := SaveConfig(cfgFile, cfg); err != nil {
				return err
			}
			fmt.Printf("Saved config at %s\n", cfgFile)
			// Create state if missing
			if _, err := os.Stat(stateFile); os.IsNotExist(err) {
				if err := SaveState(stateFile, make(State)); err != nil {
					return err
				}
				fmt.Printf("Created default state at %s\n", stateFile)
			}
			return nil
		},
	}

	// update
	update := &cobra.Command{
		Use:     "update [modpack]",
		Aliases: []string{"update-pack", "upd"},
		Short:   "Check & download new versions for a modpack",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packName := args[0]
			cfg, err := LoadConfig(cfgFile)
			if err != nil {
				return err
			}
			packCfg, ok := cfg.Modpacks[packName]
			if !ok {
				return fmt.Errorf("modpack %q not found", packName)
			}

			// Use pack-specific version and loader
			gameVersion := packCfg.MCVersion
			loader := packCfg.Loader

			// Allow overriding via flags (optional, keep or remove this logic)
			if mcVersionFlag != "" {
				fmt.Printf("Overriding MC version for %s: %s -> %s\n", packName, gameVersion, mcVersionFlag)
				gameVersion = mcVersionFlag
			}
			if loaderFlag != "" {
				fmt.Printf("Overriding loader for %s: %s -> %s\n", packName, loader, loaderFlag)
				loader = loaderFlag
			}

			state, err := LoadState(stateFile)
			if err != nil {
				return err
			}
			if state[packName] == nil {
				state[packName] = make(map[string]ModState)
			}
			reader := bufio.NewReader(os.Stdin)
			packState := state[packName]
			needsSave := false

			for _, slug := range packCfg.Mods {
				fmt.Printf("\nChecking %s...\n", slug) // Simplified initial message

				modState, modInState := packState[slug]
				fileExists := false
				destDir := filepath.Join(modsDir, packName)
				expectedFilePath := ""
				if modInState && modState.Filename != "" {
					expectedFilePath = filepath.Join(destDir, modState.Filename)
					if _, err := os.Stat(expectedFilePath); err == nil {
						fileExists = true
					} else if !os.IsNotExist(err) {
						fmt.Printf("  ✗ Error checking file %s: %v\n", expectedFilePath, err)
						// Decide whether to continue or skip based on error? For now, continue.
					}
				}

				ver, err := FetchLatestVersion(slug, gameVersion, loader)
				if err != nil {
					fmt.Printf("  ✗ Error fetching latest version: %v\n", err)
					continue
				}

				// Determine reason for action
				// downloadReason := "" // "new", "version-update", "missing-file" // Not strictly needed now
				promptMessage := ""
				needsDownload := false

				if !modInState {
					needsDownload = true
					promptMessage = fmt.Sprintf("  + New mod found: %s (Version: %s). Download?", slug, ver.ID)
				} else if ver.ID != modState.VersionID {
					needsDownload = true
					promptMessage = fmt.Sprintf("  ⚠ Update available: %s (%s -> %s). Update?", slug, modState.VersionID, ver.ID)
				} else if !fileExists {
					needsDownload = true
					// Slightly different message if filename was known vs unknown (old state format)
					if modState.Filename != "" {
						promptMessage = fmt.Sprintf("  ! File missing: %s (Version: %s). Redownload?", slug, ver.ID)
					} else {
						promptMessage = fmt.Sprintf("  ! File needed: %s (Version: %s). Download?", slug, ver.ID)
					}
				} else {
					// Up to date and file exists
					fmt.Printf("  ✓ Up to date (%s)\n", ver.ID)
					continue // Skip to next mod
				}

				// Ask user if needed
				proceed := autoYes
				if needsDownload && !proceed { // Only prompt if a download is actually needed
					fmt.Print(promptMessage + " (y/N) ")
					yn, _ := reader.ReadString('\n')
					yn = strings.TrimSpace(strings.ToLower(yn))
					if yn == "y" {
						proceed = true
					}
				}

				if !proceed {
					fmt.Println("    Skipped.")
					continue
				}

				// --- Perform Download --- 

				// Remove old file ONLY if it exists AND the new filename is different
				if fileExists && expectedFilePath != "" && modState.Filename != ver.Files[0].Filename {
					if verbose {
						fmt.Printf("    Removing old file: %s\n", expectedFilePath)
					}
					if err := os.Remove(expectedFilePath); err != nil {
						fmt.Printf("    ✗ Failed to remove old file: %v\n", err)
						// Continue anyway, maybe download will overwrite or fail
					}
				}

				if verbose {
					fmt.Printf("    Ensuring directory %s exists\n", destDir)
				}
				if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
					fmt.Printf("    ✗ Failed to create directory: %v\n", err)
					continue
				}

				// Assuming the first file is the correct one
				if len(ver.Files) == 0 {
					fmt.Printf("    ✗ No files found for version %s\n", ver.ID)
					continue
				}
				downloadURL := ver.Files[0].URL
				expectedFilename := ver.Files[0].Filename

				fmt.Printf("    Downloading %s...\n", expectedFilename)
				outPath, err := DownloadFile(downloadURL, destDir)
				if err != nil {
					fmt.Printf("    ✗ Download failed: %v\n", err)
					continue
				}
				fmt.Printf("    ✓ Downloaded: %s\n", filepath.Base(outPath))

				// Update state with new version ID and filename
				packState[slug] = ModState{VersionID: ver.ID, Filename: filepath.Base(outPath)}
				needsSave = true

			} // End loop through mods

			if needsSave {
				if err := SaveState(stateFile, state); err != nil {
					return err
				}
			}
			fmt.Println("\nUpdate check complete.")
			return nil
		},
	}

	// check-updates
	checkUpdatesCmd := &cobra.Command{
		Use:   "check-updates [modpack]", // Renamed from "status"
		Short: "Check Modrinth for newer versions of mods in a modpack", // Updated description
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packName := args[0]
			cfg, err := LoadConfig(cfgFile)
			if err != nil {
				return err
			}
			packCfg, ok := cfg.Modpacks[packName]
			if !ok {
				return fmt.Errorf("modpack %q not found", packName)
			}
			state, err := LoadState(stateFile)
			if err != nil {
				return err
			}
			if state[packName] == nil {
				state[packName] = make(map[string]ModState)
			}

			// Use pack-specific version and loader
			gameVersion := packCfg.MCVersion
			loader := packCfg.Loader
			// Allow overriding via flags (optional)
			if mcVersionFlag != "" {
				gameVersion = mcVersionFlag
			}
			if loaderFlag != "" {
				loader = loaderFlag
			}

			fmt.Printf("Checking for updates in %s (MC: %s, Loader: %s):\n", packName, gameVersion, loader)
			updatesFound := 0
			missingFiles := 0
			packState := state[packName]
			destDir := filepath.Join(modsDir, packName)

			for _, slug := range packCfg.Mods {
				modState, modInState := packState[slug]
				fileExists := false
				if modInState && modState.Filename != "" {
					filePath := filepath.Join(destDir, modState.Filename)
					if _, err := os.Stat(filePath); err == nil {
						fileExists = true
					} else if !os.IsNotExist(err) {
						fmt.Printf("  ! %s: error checking file %s: %v\n", slug, filePath, err)
					}
				}

				ver, err := FetchLatestVersion(slug, gameVersion, loader)
				if err != nil {
					fmt.Printf("  ✗ %s: error fetching version: %v\n", slug, err)
					continue
				}

				if !modInState {
					fmt.Printf("  + %s: new mod, latest version is %s\n", slug, ver.ID)
					updatesFound++ // Count as needing update
				} else if ver.ID != modState.VersionID {
					fmt.Printf("  ⚠ %s: outdated: %s → %s%s\n", slug, modState.VersionID, ver.ID, ternary(fileExists, "", " (file missing!)"))
					updatesFound++
					if !fileExists {
						missingFiles++
					}
				} else if !fileExists {
					fmt.Printf("  ! %s: file missing for current version %s\n", slug, ver.ID)
					missingFiles++
					updatesFound++ // Count as needing update because file is missing
				} else {
					if verbose {
						fmt.Printf("  ✓ %s: up to date (%s)\n", slug, ver.ID)
					}
				}
			}
			if updatesFound == 0 && missingFiles == 0 {
				fmt.Println("\nAll mods are up to date and present.")
			} else {
				fmt.Printf("\nFound %d potential update(s) and %d missing file(s). Run 'modpilot update %s' to fix.\n", updatesFound, missingFiles, packName)
			}
			return nil
		},
	}

	// sync
	syncCmd := &cobra.Command{
		Use:     "sync [modpack]",
		Aliases: []string{"sync-pack", "clean"},
		Short:   "Remove mod jars not listed in the state file for the modpack", // Updated description
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packName := args[0]
			// No need to load config for sync, only state
			state, err := LoadState(stateFile)
			if err != nil {
				return fmt.Errorf("failed to load state for sync: %w", err)
			}
			packState, ok := state[packName]
			if !ok {
				fmt.Printf("No state found for modpack %q, nothing to sync against.\n", packName)
				// Check if the directory exists anyway, maybe it needs cleaning from a previous run
				packState = make(map[string]ModState) // Treat as empty state
			}

			dir := filepath.Join(modsDir, packName)
			files, err := ioutil.ReadDir(dir)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Printf("Mods directory for %s (%s) does not exist, nothing to sync.\n", packName, dir)
					return nil // Not an error if dir doesn't exist
				}
				return fmt.Errorf("failed to read mods directory %s: %w", dir, err)
			}

			// Build a map of expected filenames from the state
			expectedFiles := make(map[string]bool)
			for _, modState := range packState {
				if modState.Filename != "" {
					expectedFiles[modState.Filename] = true
				}
			}

			// Iterate through files in the directory
			removedCount := 0
			for _, f := range files {
				if f.IsDir() || !strings.HasSuffix(strings.ToLower(f.Name()), ".jar") {
					continue // Skip directories and non-jar files
				}

				if !expectedFiles[f.Name()] {
					filePath := filepath.Join(dir, f.Name())
					fmt.Printf("Removing %s (not found in state for %s)...\n", filePath, packName)
					if err := os.Remove(filePath); err != nil {
						fmt.Printf("  ✗ Failed to remove: %v\n", err)
					} else {
						removedCount++
					}
				}
			}
			if removedCount > 0 {
				fmt.Printf("Sync complete. Removed %d unexpected file(s).\n", removedCount)
			} else {
				fmt.Println("Sync complete. No unexpected files found.")
			}
			return nil
		},
	}

	root.AddCommand(
		listPacks,
		listMods,
		addMod,
		removeMod,
		createPack,
		deletePack,
		// setMC, // Removed
		// setLoader, // Removed
		initCmd,
		update,
		checkUpdatesCmd,
		syncCmd,
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Helper for conditional printing in check-updates
func ternary(condition bool, trueVal, falseVal string) string {
	if condition {
		return trueVal
	}
	return falseVal
}
