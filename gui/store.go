package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
)

type storedProfile struct {
	Site      string `json:"site,omitempty"`
	Login     string `json:"login"`
	Counter   uint64 `json:"counter,omitempty"`
	Length    int    `json:"length,omitempty"`
	Lowercase bool   `json:"lowercase"`
	Uppercase bool   `json:"uppercase"`
	Digits    bool   `json:"digits"`
	Symbols   bool   `json:"symbols"`
	Exclude   string `json:"exclude,omitempty"`
}

type guiSettings struct {
	AutoCopy        bool `json:"auto_copy"`
	ClearAfter      int  `json:"clear_after_seconds"`
	ShowFingerprint bool `json:"show_fingerprint"`
}

type guiConfig struct {
	Settings guiSettings              `json:"settings"`
	Profiles map[string]storedProfile `json:"profiles"`
}

type profileEntry struct {
	Name string
	Data storedProfile
}

func defaultGUISettings() guiSettings {
	return guiSettings{ClearAfter: 20, ShowFingerprint: true}
}

func configFilePath() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "pass", "profiles.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "pass", "profiles.json")
}

func loadGUIConfig() (guiConfig, error) {
	cfg := guiConfig{Settings: defaultGUISettings(), Profiles: map[string]storedProfile{}}
	path := configFilePath()
	if path == "" {
		return cfg, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if len(raw) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, err
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]storedProfile{}
	}
	if cfg.Settings.ClearAfter < 0 {
		cfg.Settings.ClearAfter = 0
	}
	if cfg.Settings.ClearAfter == 0 && !cfg.Settings.AutoCopy {
		cfg.Settings.ClearAfter = 20
	}
	return cfg, nil
}

func saveGUIConfig(cfg guiConfig) error {
	path := configFilePath()
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o600)
}

func sortedProfiles(profiles map[string]storedProfile) []profileEntry {
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]profileEntry, 0, len(names))
	for _, name := range names {
		out = append(out, profileEntry{Name: name, Data: profiles[name]})
	}
	return out
}
