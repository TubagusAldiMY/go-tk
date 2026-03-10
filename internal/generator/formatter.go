package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"os/exec"
	"path/filepath"
)

// FormatGo applies gofmt formatting to Go source bytes.
// Returns the original content unchanged if formatting fails, so generation
// still succeeds even when the template produces slightly imperfect Go.
func FormatGo(src []byte) ([]byte, error) {
	formatted, err := format.Source(src)
	if err != nil {
		return src, fmt.Errorf("gofmt: %w", err)
	}
	return formatted, nil
}

// RunGoimports runs goimports on the file at path to fix/add import statements.
// Silently succeeds if goimports is not installed (it's a best-effort operation).
func RunGoimports(path string) error {
	if _, err := exec.LookPath("goimports"); err != nil {
		// goimports not installed — skip gracefully
		return nil
	}

	cmd := exec.Command("goimports", "-w", path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("goimports on %s: %w — %s", filepath.Base(path), err, stderr.String())
	}
	return nil
}

// IsGoFile reports whether a file path looks like a Go source file.
func IsGoFile(path string) bool {
	return filepath.Ext(path) == ".go"
}
