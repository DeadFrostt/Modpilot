package main

import (
    "bufio"
    "fmt"
    "io/ioutil"
    "log"
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
        Use:   "modpm",
        Short: "modpm — a Modrinth modpack manager",
        Long:  "Define modpack “stacks” in config.json, then list, add, remove, or update mods via the CLI.",
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
        Use:   "list-packs",
        Short: "List all defined modpacks",
        RunE: func(cmd *cobra.Command, args []string) error {
            cfg, err := LoadConfig(cfgFile)
            if err != nil {
                return err
            }
            fmt.Println("Modpacks:")
            for name := range cfg.Modpacks {
                fmt.Printf(" • %s\n", name)
            }
            return nil
        },
    }

    // list-mods
    listMods := &cobra.Command{
        Use:   "list-mods [modpack]",
        Short: "List all mods in a modpack",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            pack := args[0]
            cfg, err := LoadConfig(cfgFile)
            if err != nil {
                return err
            }
            mods, ok := cfg.Modpacks[pack]
            if !ok {
                return fmt.Errorf("modpack %q not found", pack)
            }
            fmt.Printf("Mods in %s:\n", pack)
            for _, slug := range mods {
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
            pack := args[0]
            slugs := args[1:]
            cfg, err := LoadConfig(cfgFile)
            if err != nil {
                return err
            }
            mods, ok := cfg.Modpacks[pack]
            if !ok {
                return fmt.Errorf("modpack %q not found", pack)
            }
            changed := false
            for _, slug := range slugs {
                exists := false
                for _, m := range mods {
                    if m == slug {
                        fmt.Printf("%q already in %s\n", slug, pack)
                        exists = true
                        break
                    }
                }
                if !exists {
                    mods = append(mods, slug)
                    fmt.Printf("Added %q to %s\n", slug, pack)
                    changed = true
                }
            }
            if changed {
                cfg.Modpacks[pack] = mods
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
            pack := args[0]
            rem := args[1:]
            cfg, err := LoadConfig(cfgFile)
            if err != nil {
                return err
            }
            mods, ok := cfg.Modpacks[pack]
            if !ok {
                return fmt.Errorf("modpack %q not found", pack)
            }
            orig := mods
            for _, slug := range rem {
                found := false
                newList := make([]string, 0, len(mods))
                for _, m := range mods {
                    if m == slug {
                        found = true
                    } else {
                        newList = append(newList, m)
                    }
                }
                if !found {
                    fmt.Printf("%q not in %s\n", slug, pack)
                } else {
                    mods = newList
                    fmt.Printf("Removed %q from %s\n", slug, pack)
                }
            }
            if len(mods) != len(orig) {
                cfg.Modpacks[pack] = mods
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
        Short: "Create a new modpack in the config",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            name := args[0]
            cfg, err := LoadConfig(cfgFile)
            if err != nil {
                return err
            }
            if _, exists := cfg.Modpacks[name]; exists {
                fmt.Printf("modpack %q already exists\n", name)
                return nil
            }
            cfg.Modpacks[name] = []string{}
            if err := SaveConfig(cfgFile, cfg); err != nil {
                return err
            }
            fmt.Printf("Created modpack %q\n", name)
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
            return nil
        },
    }

    // set-mc-version
    setMC := &cobra.Command{
        Use:   "set-mc [version]",
        Short: "Set default Minecraft version in config",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            version := args[0]
            cfg, err := LoadConfig(cfgFile)
            if err != nil {
                return err
            }
            cfg.MCVersion = version
            if err := SaveConfig(cfgFile, cfg); err != nil {
                return err
            }
            fmt.Printf("Set mc_version to %q\n", version)
            return nil
        },
    }

    // set-loader
    setLoader := &cobra.Command{
        Use:   "set-loader [loader]",
        Short: "Set default mod loader in config",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            loader := args[0]
            cfg, err := LoadConfig(cfgFile)
            if err != nil {
                return err
            }
            cfg.Loader = loader
            if err := SaveConfig(cfgFile, cfg); err != nil {
                return err
            }
            fmt.Printf("Set loader to %q\n", loader)
            return nil
        },
    }

    // init
    initCmd := &cobra.Command{
        Use:   "init",
        Short: "Initialize config and state files",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Load or create config
            var cfg *Config
            if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
                fmt.Printf("Creating new config at %s\n", cfgFile)
                cfg = &Config{MCVersion: "", Loader: "", Modpacks: make(map[string][]string)}
            } else {
                var err error
                cfg, err = LoadConfig(cfgFile)
                if err != nil {
                    return err
                }
            }
            reader := bufio.NewReader(os.Stdin)
            if cfg.MCVersion == "" {
                fmt.Print("Enter default Minecraft version (e.g. 1.19.2): ")
                v, _ := reader.ReadString('\n')
                cfg.MCVersion = strings.TrimSpace(v)
            }
            if cfg.Loader == "" {
                fmt.Print("Enter default mod loader (fabric/forge/...): ")
                l, _ := reader.ReadString('\n')
                cfg.Loader = strings.TrimSpace(l)
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
        Use:   "update [modpack]",
        Short: "Check & download new versions for a modpack",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            pack := args[0]
            cfg, err := LoadConfig(cfgFile)
            if err != nil {
                return err
            }
            slugs, ok := cfg.Modpacks[pack]
            if !ok {
                return fmt.Errorf("modpack %q not found", pack)
            }

            // Determine which MC version & loader to use
            gameVersion := cfg.MCVersion
            if mcVersionFlag != "" {
                gameVersion = mcVersionFlag
            }
            loader := cfg.Loader
            if loaderFlag != "" {
                loader = loaderFlag
            }

            state, err := LoadState(stateFile)
            if err != nil {
                return err
            }
            if state[pack] == nil {
                state[pack] = make(map[string]string)
            }
            reader := bufio.NewReader(os.Stdin)

            for _, slug := range slugs {
                fmt.Printf("\nChecking %s for %s/%s…\n", slug, gameVersion, loader)
                ver, err := FetchLatestVersion(slug, gameVersion, loader)
                if err != nil {
                    fmt.Printf("  ✗ fetch error: %v\n", err)
                    continue
                }
                last := state[pack][slug]
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
                destDir := filepath.Join(modsDir, pack)
                if verbose {
                    fmt.Printf("Ensuring directory %s exists\n", destDir)
                }
                if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
                    fmt.Printf("    ✗ mkdir failed: %v\n", err)
                    continue
                }
                outPath, err := DownloadFile(ver.Files[0].URL, fmt.Sprintf("%s/%s", modsDir, pack))
                if err != nil {
                    fmt.Printf("    ✗ download failed: %v\n", err)
                    continue
                }
                fmt.Printf("    ✓ downloaded to %s\n", outPath)
                state[pack][slug] = ver.ID
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
            pack := args[0]
            cfg, err := LoadConfig(cfgFile)
            if err != nil {
                return err
            }
            slugs, ok := cfg.Modpacks[pack]
            if !ok {
                return fmt.Errorf("modpack %q not found", pack)
            }
            state, err := LoadState(stateFile)
            if err != nil {
                return err
            }
            if state[pack] == nil {
                state[pack] = make(map[string]string)
            }
            for _, slug := range slugs {
                fmt.Printf("\n[%s] checking %s...\n", pack, slug)
                ver, err := FetchLatestVersion(slug, cfg.MCVersion, cfg.Loader)
                if err != nil {
                    fmt.Printf("  ✗ error: %v\n", err)
                    continue
                }
                last := state[pack][slug]
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
        Use:   "sync [modpack]",
        Short: "Remove jars not listed in the modpack config",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            pack := args[0]
            cfg, err := LoadConfig(cfgFile)
            if err != nil {
                return err
            }
            slugs, ok := cfg.Modpacks[pack]
            if !ok {
                return fmt.Errorf("modpack %q not found", pack)
            }
            dir := filepath.Join(modsDir, pack)
            files, err := ioutil.ReadDir(dir)
            if err != nil {
                return err
            }
            keep := make(map[string]struct{})
            for _, slug := range slugs {
                keep[slug] = struct{}{}
            }
            for _, f := range files {
                if f.IsDir() {
                    continue
                }
                name := f.Name()
                match := false
                for slug := range keep {
                    if strings.HasPrefix(name, slug) {
                        match = true
                        break
                    }
                }
                if !match {
                    path := filepath.Join(dir, name)
                    if verbose {
                        fmt.Printf("Removing: %s\n", path)
                    }
                    if err := os.Remove(path); err != nil {
                        fmt.Printf("Failed to remove %s: %v\n", path, err)
                    }
                }
            }
            fmt.Println("Sync complete.")
            return nil
        },
    }

    // Register subcommands
    root.AddCommand(listPacks, listMods, addMod, removeMod, createPack,
        deletePack, setMC, setLoader, initCmd, update, statusCmd, syncCmd)

    if err := root.Execute(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
