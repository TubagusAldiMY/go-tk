// Package generator provides template rendering and atomic file operations.
//
// This file implements safe file writing to prevent corruption during
// concurrent operations or power failures. The atomic write pattern
// (temp + rename) is POSIX-compliant and the gold standard for safe writes.
//
// Why atomic writes matter:
//   - Prevents partial writes if process is killed mid-operation
//   - Safe for concurrent reads (readers see old or new, never partial)
//   - No window where file is empty or corrupted
//
// All generated code (project scaffolding, CRUD, etc.) MUST use WriteAtomic()
// to maintain idempotency guarantees (running same command twice is safe).
package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteAtomic writes content to path using a temp-file-then-rename strategy
// to ensure atomicity. Creates parent directories as needed.
//
// Algorithm:
//   1. Create parent directories (os.MkdirAll)
//   2. Create temp file in same directory as target (same filesystem = atomic rename)
//   3. Write content to temp file
//   4. Set permissions on temp file
//   5. Close temp file
//   6. Rename temp → target (atomic operation on POSIX)
//   7. Cleanup temp file if any step fails
//
// Parameters:
//   path    — Target file path (absolute or relative)
//   content — File content as bytes
//   perm    — Unix permissions (e.g. 0o644 for rw-r--r--)
//
// Safety guarantees:
//   - Readers never see partial content
//   - If process is killed mid-write, old file is intact
//   - If disk is full, temp file is cleaned up (no orphaned files)
//
// Example:
//   content := []byte("package main\n\nfunc main() {}\n")
//   err := WriteAtomic("cmd/server/main.go", content, 0o644)
//
// Thread-safety: Safe for concurrent writes to DIFFERENT files.
// Writes to the SAME file are serialized by the filesystem (last write wins).
func WriteAtomic(path string, content []byte, perm os.FileMode) error {
	// Ensure parent directories exist (0o755 = rwxr-xr-x)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directories for %s: %w", path, err)
	}

	// Create temp file in same directory (ensures same filesystem for atomic rename).
	// Pattern: .gotk-tmp-* → hidden, recognizable, cleaned up by defer.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".gotk-tmp-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()

	// Ensure cleanup on failure — if rename succeeds, Remove is no-op.
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	// Write content
	if _, err := tmp.Write(content); err != nil {
		return fmt.Errorf("writing to temp file: %w", err)
	}

	// Set permissions BEFORE rename (some systems check perms on target)
	if err := tmp.Chmod(perm); err != nil {
		return fmt.Errorf("setting permissions: %w", err)
	}

	// Close before rename (required on Windows, good practice on POSIX)
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	// Atomic rename — this is the critical operation.
	// On POSIX: atomic if src and dst are on same filesystem.
	// On Windows: may fail if target exists and is open (rare in our case).
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("renaming temp file to %s: %w", path, err)
	}

	return nil
}

// FileExists reports whether path exists on the filesystem.
//
// This uses os.Stat() and checks for any error. Returns false for:
//   - File does not exist (os.ErrNotExist)
//   - Permission denied
//   - Path is a broken symlink
//
// Use this to check existence before --force overwrites.
// Do NOT use this for TOCTOU-sensitive operations (use atomic writes instead).
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// EnsureDir creates directory and all parents, no-op if already exists.
//
// Permissions: 0o755 (rwxr-xr-x) — owner can write, others can read+execute.
// This is safe to call multiple times (idempotent).
//
// Use this before writing files to new directories (though WriteAtomic()
// already calls MkdirAll internally, so usually not needed).
func EnsureDir(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", path, err)
	}
	return nil
}
