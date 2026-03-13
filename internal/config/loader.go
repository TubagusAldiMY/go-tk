// Package config provides gotk.yaml discovery and loading.
//
// gotk.yaml is the project-level configuration file that defines:
//   - Stack: framework (gin/fiber), database (postgres/mysql), ORM (gorm/sqlc)
//   - Paths: where to generate entities, handlers, repos, migrations
//   - Generate options: soft delete, timestamps, etc.
//
// Discovery algorithm:
// Load() walks UP the directory tree from startDir to filesystem root,
// looking for gotk.yaml. This allows go-tk commands to be run from any
// subdirectory of a generated project (similar to git's .git discovery).
//
// Example project structure:
//
//	/project-root/
//	  gotk.yaml              ← Found by Load() from any subdir
//	  internal/
//	    domain/
//	    infrastructure/
//
// If gotk.yaml is not found, commands that require it (gen, migrate, test, analyze)
// will fail with ErrConfigNotFound, prompting the user to run "go-tk new" first.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// ConfigFileName is the expected name of the project configuration file.
// This MUST be present at the project root for go-tk commands to work.
const ConfigFileName = "gotk.yaml"

// ErrConfigNotFound is returned when gotk.yaml cannot be located.
// This typically means the user is not in a go-tk project directory.
var ErrConfigNotFound = errors.New("gotk.yaml not found; run 'go-tk new' to create a project first")

// Load searches for gotk.yaml starting from startDir, walking up to the
// filesystem root. Returns the parsed Config on success.
//
// Discovery algorithm:
//  1. Start at startDir (typically os.Getwd())
//  2. Check if gotk.yaml exists in current directory
//  3. If not found, move to parent directory
//  4. Repeat until found or filesystem root reached
//  5. Return ErrConfigNotFound if not found
//
// This allows go-tk commands to be run from ANY subdirectory of a project,
// similar to how git commands work from any subdir of a repo.
//
// Example:
//
//	cfg, err := config.Load(cwd)
//	if err != nil {
//	    return fmt.Errorf("not in a go-tk project: %w", err)
//	}
//
// Error cases:
//   - gotk.yaml not found → ErrConfigNotFound
//   - gotk.yaml invalid YAML → viper parse error
//   - gotk.yaml missing required fields → unmarshal error
func Load(startDir string) (*Config, error) {
	configDir, err := findConfigDir(startDir)
	if err != nil {
		return nil, err
	}
	return loadFromExactPath(filepath.Join(configDir, ConfigFileName))
}

// LoadFromDir loads gotk.yaml from a specific directory without parent search.
//
// Use this when you know the exact project root (e.g. during "go-tk new"
// when creating gotk.yaml in a new directory).
//
// Use Load() for normal operation (auto-discovery from any subdir).
func LoadFromDir(dir string) (*Config, error) {
	return loadFromExactPath(filepath.Join(dir, ConfigFileName))
}

// loadFromExactPath parses gotk.yaml from a specific file path.
//
// Uses Viper for YAML parsing because:
//   - Handles environment variable substitution (future feature)
//   - Provides better error messages than yaml.Unmarshal
//   - Consistent with Viper usage in generated projects
//
// This is an internal helper — callers should use Load() or LoadFromDir().
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
//
// Algorithm:
//
//	current = abs(startDir)
//	loop:
//	  if gotk.yaml exists in current → return current
//	  parent = dirname(current)
//	  if parent == current → reached root, break
//	  current = parent
//	return ErrConfigNotFound
//
// This is the "git-style" discovery pattern used by many dev tools.
func findConfigDir(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("resolving directory: %w", err)
	}

	for {
		candidate := filepath.Join(dir, ConfigFileName)
		if _, err := os.Stat(candidate); err == nil {
			return dir, nil // Found!
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root (parent == self)
			break
		}
		dir = parent
	}

	return "", ErrConfigNotFound
}
