package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config stores global CLI preferences.
type Config struct {
	Theme          string `json:"theme,omitempty"`
	LastLoginEmail string `json:"last_login_email,omitempty"`
	// UserMirrors are HTTPS origins the user trusts with their login in
	// addition to the built-in Z-Library domains.
	UserMirrors []string `json:"user_mirrors,omitempty"`
}

func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "zlib", "config.json")
}

func LoadConfig() (Config, error) {
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return Config{}, err
	}
	var c Config
	return c, json.Unmarshal(data, &c)
}

func SaveConfig(c Config) error {
	path := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
