package checks

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/TubagusAldiMY/go-tk/internal/command/analyze/types"
)

// CheckCircularImports detects import cycles among internal packages in the project.
//
// Algorithm:
//  1. Read the module path from go.mod
//  2. Walk all non-test .go files under internal/
//  3. Build an import graph: package → []imported_internal_packages
//  4. DFS cycle detection (Tarjan-style coloring)
//  5. Report each unique cycle once (CRITICAL severity)
func CheckCircularImports(projectDir string) ([]types.Issue, int, error) {
	modulePath, err := readModulePath(projectDir)
	if err != nil {
		// Not a Go module or go.mod unreadable — skip silently.
		return nil, 0, nil
	}

	internalDir := filepath.Join(projectDir, "internal")

	// Map: short package path (relative to module root) → []imports
	graph := make(map[string][]string)
	scanned := 0

	walkErr := filepath.WalkDir(internalDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || isTestFile(path) {
			return nil
		}
		scanned++

		pkgPath := dirToShortPkg(filepath.Dir(path), projectDir, modulePath)
		imports := parseInternalImports(path, modulePath)

		existing := graph[pkgPath]
		for _, imp := range imports {
			short := strings.TrimPrefix(imp, modulePath+"/")
			if !containsStr(existing, short) {
				existing = append(existing, short)
			}
		}
		graph[pkgPath] = existing

		return nil
	})
	if walkErr != nil {
		return nil, scanned, nil
	}

	issues := detectCycles(graph)
	return issues, scanned, nil
}

// detectCycles runs DFS over the import graph and returns one Issue per unique cycle.
func detectCycles(graph map[string][]string) []types.Issue {
	type color int
	const (
		white color = iota // unvisited
		gray               // in current DFS path
		black              // fully processed
	)

	colorOf := make(map[string]color)
	parent := make(map[string]string)
	var issues []types.Issue
	reported := make(map[string]bool)

	// Stable iteration order for deterministic output.
	pkgs := make([]string, 0, len(graph))
	for k := range graph {
		pkgs = append(pkgs, k)
	}
	sort.Strings(pkgs)

	var dfs func(pkg string)
	dfs = func(pkg string) {
		colorOf[pkg] = gray

		for _, dep := range graph[pkg] {
			switch colorOf[dep] {
			case white:
				parent[dep] = pkg
				dfs(dep)
			case gray:
				// Back-edge → cycle found; reconstruct path.
				cycle := reconstructCycle(parent, dep, pkg)
				key := cycleKey(cycle)
				if !reported[key] {
					reported[key] = true
					issues = append(issues, types.Issue{
						Kind:     types.KindCircularImport,
						Severity: types.SeverityCritical,
						File:     dep,
						Message:  fmt.Sprintf("import cycle detected: %s", strings.Join(cycle, " → ")),
					})
				}
			}
		}

		colorOf[pkg] = black
	}

	for _, pkg := range pkgs {
		if colorOf[pkg] == white {
			dfs(pkg)
		}
	}

	return issues
}

// reconstructCycle rebuilds the cycle path from the parent map.
// cycleStart is the node where the back-edge lands.
func reconstructCycle(parent map[string]string, cycleStart, current string) []string {
	path := []string{cycleStart}
	node := current
	for node != cycleStart {
		path = append([]string{node}, path...)
		p, ok := parent[node]
		if !ok {
			break
		}
		node = p
	}
	path = append(path, cycleStart) // close the loop
	return path
}

// cycleKey returns a canonical string key for a cycle (for deduplication).
// Rotates the cycle to start from its lexicographically smallest element.
func cycleKey(cycle []string) string {
	if len(cycle) == 0 {
		return ""
	}
	// Drop the repeated last element (closing node).
	c := cycle[:len(cycle)-1]
	if len(c) == 0 {
		return ""
	}

	minIdx := 0
	for i, s := range c {
		if s < c[minIdx] {
			minIdx = i
		}
	}
	rotated := append(c[minIdx:], c[:minIdx]...)
	return strings.Join(rotated, "→")
}

// readModulePath parses the module path from go.mod in the given directory.
func readModulePath(projectDir string) (string, error) {
	f, err := os.Open(filepath.Join(projectDir, "go.mod"))
	if err != nil {
		return "", err
	}
	defer f.Close() //nolint:errcheck

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module "), nil
		}
	}
	return "", fmt.Errorf("module directive not found in go.mod")
}

// dirToShortPkg converts an absolute directory path to a short package path
// relative to the module root (e.g. "internal/domain/entity").
func dirToShortPkg(absDir, projectDir, modulePath string) string {
	rel, err := filepath.Rel(projectDir, absDir)
	if err != nil {
		return absDir
	}
	// Normalize to forward slashes.
	return filepath.ToSlash(rel)
}

// parseInternalImports returns all imports from the file that belong to the same module.
// It handles all valid Go import forms:
//   - import "path"
//   - import alias "path"
//   - import ( "path" ) blocks with optional aliases
func parseInternalImports(filePath, modulePath string) []string {
	f, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer f.Close() //nolint:errcheck

	var imports []string
	inImport := false
	prefix := modulePath + "/"

	addIfInternal := func(line string) {
		// Extract the quoted import path from a line that may have an alias.
		// e.g. `_ "github.com/foo/bar"` or `foo "github.com/foo/bar"` or `"github.com/foo/bar"`
		start := strings.Index(line, `"`)
		if start < 0 {
			return
		}
		end := strings.LastIndex(line, `"`)
		if end <= start {
			return
		}
		imp := line[start+1 : end]
		if strings.HasPrefix(imp, prefix) {
			imports = append(imports, imp)
		}
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "import (" {
			inImport = true
			continue
		}
		if inImport && line == ")" {
			inImport = false
			continue
		}

		// Single-line import (with or without alias): import ["alias"] "path"
		if strings.HasPrefix(line, "import ") {
			addIfInternal(line)
			continue
		}

		if inImport {
			addIfInternal(line)
		}
	}

	return imports
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
