package main

import (
    "encoding/json"
    "io/ioutil"
    "os"
)

// ModpackConfig holds settings specific to a single modpack
type ModpackConfig struct {
	MCVersion string   `json:"mc_version"`
	Loader    string   `json:"loader"`
	Mods      []string `json:"mods"`
}

type Config struct {
	// Optional global defaults for new packs
	DefaultMCVersion string `json:"default_mc_version,omitempty"`
	DefaultLoader    string `json:"default_loader,omitempty"`

	// Modpacks maps pack name to its specific configuration
	Modpacks map[string]ModpackConfig `json:"modpacks"`
}

// State keeps track of last-downloaded version IDs
// State[packName][modSlug] = versionID
type State map[string]map[string]string

func LoadConfig(path string) (*Config, error) {
    b, err := ioutil.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var cfg Config
    if err := json.Unmarshal(b, &cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}

func SaveConfig(path string, cfg *Config) error {
    b, err := json.MarshalIndent(cfg, "", "  ")
    if err != nil {
        return err
    }
    return ioutil.WriteFile(path, b, 0644)
}

func LoadState(path string) (State, error) {
    if _, err := os.Stat(path); os.IsNotExist(err) {
        return make(State), nil
    }
    b, err := ioutil.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var s State
    if err := json.Unmarshal(b, &s); err != nil {
        return nil, err
    }
    return s, nil
}

func SaveState(path string, s State) error {
    b, err := json.MarshalIndent(s, "", "  ")
    if err != nil {
        return err
    }
    return ioutil.WriteFile(path, b, 0644)
}
