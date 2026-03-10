package checks

import (
	"path/filepath"
	"strings"
)

// isTestFile returns true for _test.go files.
func isTestFile(path string) bool {
	return strings.HasSuffix(path, "_test.go")
}

// shortenPath returns a path relative to the current working directory,
// or the base name if shortening fails.
func shortenPath(path string) string {
	abs, err := filepath.Abs(".")
	if err != nil {
		return filepath.Base(path)
	}
	rel, err := filepath.Rel(abs, path)
	if err != nil {
		return filepath.Base(path)
	}
	return rel
}
