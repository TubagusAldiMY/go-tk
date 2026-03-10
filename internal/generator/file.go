package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteAtomic writes content to path using a temp-file-then-rename strategy
// to ensure atomicity. Creates parent directories as needed.
func WriteAtomic(path string, content []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directories for %s: %w", path, err)
	}

	// Write to a temp file in the same directory (same filesystem).
	tmp, err := os.CreateTemp(filepath.Dir(path), ".gotk-tmp-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()

	// Ensure cleanup on failure.
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath) // no-op if rename succeeded
	}()

	if _, err := tmp.Write(content); err != nil {
		return fmt.Errorf("writing to temp file: %w", err)
	}
	if err := tmp.Chmod(perm); err != nil {
		return fmt.Errorf("setting permissions: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("renaming temp file to %s: %w", path, err)
	}

	return nil
}

// FileExists reports whether path exists on the filesystem.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// EnsureDir creates directory and all parents, no-op if already exists.
func EnsureDir(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", path, err)
	}
	return nil
}
