package generator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "file.txt")
	content := []byte("hello, go-tk")

	if err := WriteAtomic(path, content, 0o644); err != nil {
		t.Fatalf("WriteAtomic failed: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading written file: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", got, content)
	}
}

func TestWriteAtomicIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")

	first := []byte("first")
	second := []byte("second")

	if err := WriteAtomic(path, first, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteAtomic(path, second, 0o644); err != nil {
		t.Fatal(err)
	}

	got, _ := os.ReadFile(path)
	if string(got) != "second" {
		t.Errorf("expected second write to overwrite, got %q", got)
	}
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()

	existing := filepath.Join(dir, "exists.txt")
	_ = os.WriteFile(existing, []byte("x"), 0o644)

	if !FileExists(existing) {
		t.Error("FileExists should return true for existing file")
	}
	if FileExists(filepath.Join(dir, "missing.txt")) {
		t.Error("FileExists should return false for missing file")
	}
}

func TestEnsureDir(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "a", "b", "c")

	if err := EnsureDir(target); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected a directory")
	}

	// Calling again must not error (idempotent).
	if err := EnsureDir(target); err != nil {
		t.Errorf("EnsureDir not idempotent: %v", err)
	}
}
