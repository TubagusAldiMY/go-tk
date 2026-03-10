package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const ConfigFileName = "gotk.yaml"

// ErrConfigNotFound is returned when gotk.yaml cannot be located.
var ErrConfigNotFound = errors.New("gotk.yaml not found; run 'go-tk new' to create a project first")

// Load searches for gotk.yaml starting from startDir, walking up to the
// filesystem root. Returns the parsed Config on success.
func Load(startDir string) (*Config, error) {
	configDir, err := findConfigDir(startDir)
	if err != nil {
		return nil, err
	}
	return loadFromExactPath(filepath.Join(configDir, ConfigFileName))
}

// LoadFromDir loads gotk.yaml from a specific directory without parent search.
func LoadFromDir(dir string) (*Config, error) {
	return loadFromExactPath(filepath.Join(dir, ConfigFileName))
}

func loadFromExactPath(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	return &cfg, nil
}

// findConfigDir walks from startDir up to the root looking for ConfigFileName.
func findConfigDir(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("resolving directory: %w", err)
	}

	for {
		candidate := filepath.Join(dir, ConfigFileName)
		if _, err := os.Stat(candidate); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached filesystem root
		}
		dir = parent
	}

	return "", ErrConfigNotFound
}
