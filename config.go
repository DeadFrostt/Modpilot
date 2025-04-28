package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// ModpackConfig defines settings for a single modpack
type ModpackConfig struct {
	MCVersion string   `json:"mc_version"`
	Loader    string   `json:"loader"`
	Mods      []string `json:"mods"`
}

// Config is the top-level structure for config.json
type Config struct {
	DefaultMCVersion string                   `json:"default_mc_version,omitempty"`
	DefaultLoader    string                   `json:"default_loader,omitempty"`
	Modpacks         map[string]ModpackConfig `json:"modpacks"`
}

// ModState stores the last known version ID and filename for a mod
type ModState struct {
	VersionID string `json:"version_id"`
	Filename  string `json:"filename"`
}

// State maps modpack names to maps of mod slugs to their state
type State map[string]map[string]ModState // packName -> slug -> ModState

// LoadConfig reads and parses the config file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// --- Validation --- 
	if cfg.Modpacks == nil {
		// Allow empty config if the file exists but has no modpacks yet
		cfg.Modpacks = make(map[string]ModpackConfig)
	}

	for name, packCfg := range cfg.Modpacks {
		if packCfg.MCVersion == "" {
			return nil, fmt.Errorf("config validation failed: modpack %q is missing 'mc_version'", name)
		}
		if packCfg.Loader == "" {
			return nil, fmt.Errorf("config validation failed: modpack %q is missing 'loader'", name)
		}
		// Note: We don't validate if the version/loader combo is *correct*, just that they exist.
	}
	// --- End Validation ---

	return &cfg, nil
}

// SaveConfig writes the config structure back to the file
func SaveConfig(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadState reads and parses the state file
func LoadState(path string) (State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// If state file doesn't exist, return an empty state map
			return make(State), nil
		}
		return nil, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		// Attempt to load old format (map[string]map[string]string) for backward compatibility
		var oldState map[string]map[string]string
		if errOld := json.Unmarshal(data, &oldState); errOld == nil {
			fmt.Println("Note: Converting old state.json format. Run 'update' to populate filenames.")
			newState := make(State)
			for packName, mods := range oldState {
				newState[packName] = make(map[string]ModState)
				for slug, versionID := range mods {
					newState[packName][slug] = ModState{VersionID: versionID, Filename: ""} // Filename will be populated on next update
				}
			}
			return newState, nil
		}
		// If neither new nor old format works, return the original error
		return nil, fmt.Errorf("failed to unmarshal state file %s: %w", path, err)
	}
	return state, nil
}

// SaveState writes the state structure back to the file
func SaveState(path string, state State) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
