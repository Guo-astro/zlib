package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/heartleo/zlib/internal/zlib"
)

func KindleConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "zlib", "kindle.json")
}

func LoadKindleConfig() (zlib.KindleConfig, error) {
	return LoadKindleConfigFrom(KindleConfigPath())
}

func LoadKindleConfigFrom(path string) (zlib.KindleConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return zlib.KindleConfig{}, err
	}
	var cfg zlib.KindleConfig
	return cfg, json.Unmarshal(data, &cfg)
}

func SaveKindleConfig(cfg zlib.KindleConfig) error {
	return SaveKindleConfigTo(KindleConfigPath(), cfg)
}

func SaveKindleConfigTo(path string, cfg zlib.KindleConfig) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
