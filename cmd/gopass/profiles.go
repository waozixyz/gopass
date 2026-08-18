package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	password "github.com/waozixyz/gopass"
)

// Profile holds the saved settings for one named identity: a login plus the
// generator options and clipboard behavior to use with it. Profiles contain
// no secrets — the master password always comes from a prompt, the
// environment, or an encrypted vault referenced by path.
type Profile struct {
	Login      string `json:"login"`
	Counter    uint64 `json:"counter,omitempty"`
	Length     int    `json:"length,omitempty"`
	Lowercase  *bool  `json:"lowercase,omitempty"`
	Uppercase  *bool  `json:"uppercase,omitempty"`
	Digits     *bool  `json:"digits,omitempty"`
	Symbols    *bool  `json:"symbols,omitempty"`
	Exclude    string `json:"exclude,omitempty"`
	Vault      string `json:"vault,omitempty"`
	Copy       *bool  `json:"copy,omitempty"`
	ClearAfter string `json:"clear_after,omitempty"`
}

// Config is the optional profiles file. Every field is a default that flags
// and profile entries override; a missing file leaves gopass fully stateless.
type Config struct {
	Vault      string             `json:"vault,omitempty"`
	Copy       *bool              `json:"copy,omitempty"`
	ClearAfter string             `json:"clear_after,omitempty"`
	Profiles   map[string]Profile `json:"profiles"`
}

// configPath is a variable so tests can point it at a temporary file.
var configPath = defaultConfigPath

func defaultConfigPath() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "gopass", "profiles.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "gopass", "profiles.json")
}

// loadConfig reads the profiles file. A missing file is not an error: the
// feature is simply off and gopass behaves as a plain stateless generator.
func loadConfig() (*Config, error) {
	path := configPath()
	if path == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var config Config
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &config, nil
}

// apply fills the option fields that were not set explicitly on the command
// line, so the precedence is always flag > profile > built-in default.
func (p Profile) apply(options *password.Options, explicit map[string]bool) {
	if p.Length > 0 && !anySet(explicit, "L", "length") {
		options.Length = p.Length
	}
	if p.Counter > 0 && !anySet(explicit, "C", "counter") {
		options.Counter = p.Counter
	}
	if p.Lowercase != nil && !anySet(explicit, "l", "lowercase", "no-lowercase") {
		options.Lowercase = *p.Lowercase
	}
	if p.Uppercase != nil && !anySet(explicit, "u", "uppercase", "no-uppercase") {
		options.Uppercase = *p.Uppercase
	}
	if p.Digits != nil && !anySet(explicit, "d", "digits", "no-digits") {
		options.Digits = *p.Digits
	}
	if p.Symbols != nil && !anySet(explicit, "s", "symbols", "no-symbols") {
		options.Symbols = *p.Symbols
	}
	if p.Exclude != "" && !anySet(explicit, "exclude") {
		options.Exclude = p.Exclude
	}
}

func anySet(explicit map[string]bool, names ...string) bool {
	for _, name := range names {
		if explicit[name] {
			return true
		}
	}
	return false
}

// expandHome resolves a leading ~ so config entries stay portable.
func expandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~"))
		}
	}
	return path
}
