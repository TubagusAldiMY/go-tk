package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromDir(t *testing.T) {
	dir := t.TempDir()

	content := `version: 1
project:
  name: test-app
  module: github.com/test/test-app
stack:
  framework: gin
  database: postgres
  orm: gorm
  auth: jwt
`
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir failed: %v", err)
	}

	if cfg.Project.Name != "test-app" {
		t.Errorf("Name = %q, want %q", cfg.Project.Name, "test-app")
	}
	if cfg.Stack.Framework != "gin" {
		t.Errorf("Framework = %q, want %q", cfg.Stack.Framework, "gin")
	}
}

func TestLoadNotFound(t *testing.T) {
	dir := t.TempDir()

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error when gotk.yaml missing")
	}
	if !errors.Is(err, ErrConfigNotFound) {
		t.Errorf("expected ErrConfigNotFound, got %v", err)
	}
}

func TestLoadSearchesParent(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	content := "version: 1\nproject:\n  name: parent-app\n  module: github.com/test/parent\nstack:\n  framework: gin\n  database: postgres\n  orm: gorm\n  auth: jwt\n"
	if err := os.WriteFile(filepath.Join(root, ConfigFileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(sub) // start from deep subdirectory
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Project.Name != "parent-app" {
		t.Errorf("Name = %q, want %q", cfg.Project.Name, "parent-app")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("myapp", "github.com/user/myapp")

	if cfg.Project.Name != "myapp" {
		t.Errorf("Name = %q", cfg.Project.Name)
	}
	if cfg.Stack.Framework != FrameworkGin {
		t.Errorf("Framework = %q", cfg.Stack.Framework)
	}
	if cfg.Paths.Handlers == "" {
		t.Error("Handlers path should not be empty")
	}
}
