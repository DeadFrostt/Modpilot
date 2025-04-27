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
				state[packName] = make(map[string]string)
			}
			reader := bufio.NewReader(os.Stdin)

			for _, slug := range packCfg.Mods {
				fmt.Printf("\nChecking %s for %s/%s…\n", slug, gameVersion, loader)
				ver, err := FetchLatestVersion(slug, gameVersion, loader)
				if err != nil {
					fmt.Printf("  ✗ fetch error: %v\n", err)
					continue
				}
				last := state[packName][slug]
				if ver.ID == last {
					fmt.Printf("  ✓ up to date (%s)\n", ver.ID)
					continue
				}
				fmt.Printf("  ⚠ %s → %s\n", last, ver.ID)
				if !autoYes {
					fmt.Print("    Update? (y/N) ")
					yn, _ := reader.ReadString('\n')
					yn = strings.TrimSpace(strings.ToLower(yn))
					if yn != "y" {
						fmt.Println("    skipped.")
						continue
					}
				}
				// ensure mods directory exists
				destDir := filepath.Join(modsDir, packName)
				if verbose {
					fmt.Printf("Ensuring directory %s exists\n", destDir)
				}
				if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
					fmt.Printf("    ✗ mkdir failed: %v\n", err)
					continue
				}
				outPath, err := DownloadFile(ver.Files[0].URL, destDir)
				if err != nil {
					fmt.Printf("    ✗ download failed: %v\n", err)
					continue
				}
				fmt.Printf("    ✓ downloaded to %s\n", outPath)
				state[packName][slug] = ver.ID
			}

			if err := SaveState(stateFile, state); err != nil {
				return err
			}
			fmt.Println("\nUpdate complete.")
			return nil
		},
	}

	// status
	statusCmd := &cobra.Command{
		Use:   "status [modpack]",
		Short: "Display up-to-date/outdated status for a modpack",
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
				state[packName] = make(map[string]string)
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

			for _, slug := range packCfg.Mods {
				fmt.Printf("\n[%s] checking %s...\n", packName, slug)
				ver, err := FetchLatestVersion(slug, gameVersion, loader)
				if err != nil {
					fmt.Printf("  ✗ error: %v\n", err)
					continue
				}
				last := state[packName][slug]
				if ver.ID == last {
					fmt.Printf("  ✓ up to date (%s)\n", ver.ID)
				} else {
					fmt.Printf("  ⚠ outdated: %s → %s\n", last, ver.ID)
				}
			}
			return nil
		},
	}

	// sync
	syncCmd := &cobra.Command{
		Use:     "sync [modpack]",
		Aliases: []string{"sync-pack", "clean"},
		Short:   "Remove jars not listed in the modpack config",
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
			dir := filepath.Join(modsDir, packName)
			files, err := ioutil.ReadDir(dir)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Printf("Mods directory for %s (%s) does not exist, nothing to sync.\n", packName, dir)
					return nil // Not an error if dir doesn't exist
				}
				return err
			}

			// Build a map of expected filenames based on state
			// state, err := LoadState(stateFile) // Keep state loading commented/removed if not used
			// if err != nil {
			// 	return fmt.Errorf("failed to load state for sync: %w", err)
			// }
			// packState, ok := state[packName] // REMOVED - declared and not used
			// if !ok {
			// 	packState = make(map[string]string) // Handle case where pack has no state yet
			// }

			// Build map of slugs defined in config for this pack
			configSlugs := make(map[string]bool)
			for _, slug := range packCfg.Mods {
				configSlugs[slug] = true
			}

			// Iterate through files in the directory
			removedCount := 0
			for _, f := range files {
				if f.IsDir() {
					continue // Skip directories
				}
				// Simplistic check: Does the filename contain any known slug?
				// This is NOT robust. A better way would be to store filename in state.
				keepFile := false
				for slug := range configSlugs {
					if strings.Contains(strings.ToLower(f.Name()), strings.ToLower(slug)) {
						keepFile = true
						break
					}
				}

				if !keepFile {
					filePath := filepath.Join(dir, f.Name())
					fmt.Printf("Removing %s (not found in %s config)...\n", filePath, packName)
					if err := os.Remove(filePath); err != nil {
						fmt.Printf("  ✗ Failed to remove: %v\n", err)
					} else {
						removedCount++
					}
				}
			}
			if removedCount > 0 {
				fmt.Printf("Sync complete. Removed %d file(s).\n", removedCount)
			} else {
				fmt.Println("Sync complete. No files removed.")
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
		statusCmd,
		syncCmd,
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
