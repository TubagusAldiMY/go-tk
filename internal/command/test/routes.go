package test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TubagusAldiMY/go-tk/internal/parser"
)

// RouteInfo extends parser.Route with display helpers.
type RouteInfo struct {
	Method  string
	Path    string
	Handler string
	File    string
}

// DiscoverRoutes finds the router file in the project and extracts all routes.
// It searches for common router filenames in the interfaces/http directory.
func DiscoverRoutes(projectRoot string) ([]RouteInfo, error) {
	candidates := []string{
		filepath.Join(projectRoot, "internal", "interfaces", "http", "router.go"),
		filepath.Join(projectRoot, "internal", "router", "router.go"),
		filepath.Join(projectRoot, "router.go"),
	}

	var routerFile string
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			routerFile = c
			break
		}
	}

	// Fallback: walk internal/interfaces/http for any Go file containing route registrations.
	if routerFile == "" {
		found, err := findRouterFile(filepath.Join(projectRoot, "internal"))
		if err != nil || found == "" {
			return nil, fmt.Errorf("router file not found — run from your project root")
		}
		routerFile = found
	}

	routes, err := parser.ParseRoutesFromFile(routerFile)
	if err != nil {
		return nil, fmt.Errorf("parsing routes from %s: %w", routerFile, err)
	}

	result := make([]RouteInfo, 0, len(routes))
	for _, r := range routes {
		result = append(result, RouteInfo{
			Method:  r.Method,
			Path:    r.Path,
			Handler: r.Handler,
			File:    routerFile,
		})
	}
	return result, nil
}

// findRouterFile walks dir looking for a Go file that registers HTTP routes.
func findRouterFile(dir string) (string, error) {
	var found string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		s := string(content)
		if strings.Contains(s, ".GET(") || strings.Contains(s, ".POST(") {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found, err
}
