package checks

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/TubagusAldiMY/go-tk/internal/command/analyze/types"
	goparser "github.com/TubagusAldiMY/go-tk/internal/parser"
)

// CheckDeadRoutes cross-references route registrations with handler method declarations.
//
// It reports two finding types (per BR-06.1.1 and BR-06.1.2):
//   - Orphaned handler: exported method on a *XxxHandler type with no matching route registration (LOW)
//   - Dead route: route whose handler method name cannot be found in any handler file (HIGH)
func CheckDeadRoutes(internalDir string) ([]types.Issue, int, error) {
	scanned := 0

	// ── 1. Parse all router/app files to collect registered method names ──────
	routerFiles := findRouterFiles(internalDir)
	routedMethods := make(map[string]bool) // method name → registered?

	for _, rf := range routerFiles {
		routes, err := goparser.ParseRoutesFromFile(rf)
		if err != nil {
			continue
		}
		for _, r := range routes {
			// Handler is "varName.MethodName" — extract method name (right of last dot)
			if idx := strings.LastIndex(r.Handler, "."); idx >= 0 {
				if m := r.Handler[idx+1:]; m != "" && m != "<unknown>" {
					routedMethods[m] = true
				}
			}
		}
	}

	// ── 2. Scan handler directory for exported methods on *Handler types ──────
	handlerDir := filepath.Join(internalDir, "interfaces", "http", "handler")
	var issues []types.Issue

	walkErr := filepath.WalkDir(handlerDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || isTestFile(path) {
			return nil
		}
		scanned++
		issues = append(issues, checkOrphanedMethods(path, routedMethods)...)
		return nil
	})

	if walkErr != nil {
		// Handler dir not found — project is empty or non-standard layout; skip silently.
		return issues, scanned, nil
	}

	// ── 3. Dead routes: route methods that exist in no handler file ───────────
	// Build set of all declared handler methods.
	declaredMethods := collectDeclaredHandlerMethods(handlerDir)

	for _, rf := range routerFiles {
		routes, err := goparser.ParseRoutesFromFile(rf)
		if err != nil {
			continue
		}
		relPath := shortenPath(rf)
		for _, r := range routes {
			if idx := strings.LastIndex(r.Handler, "."); idx >= 0 {
				m := r.Handler[idx+1:]
				if m == "" || m == "<unknown>" || m == "func" {
					continue
				}
				if !declaredMethods[m] {
					issues = append(issues, types.Issue{
						Kind:     types.KindDeadRoute,
						Severity: types.SeverityHigh,
						File:     relPath,
						Message: fmt.Sprintf("route %s %s references handler method %q which is not declared in any handler file",
							r.Method, r.Path, m),
					})
				}
			}
		}
	}

	return issues, scanned, nil
}

// checkOrphanedMethods finds exported methods on *XxxHandler receiver types
// that are not referenced in any route registration.
func checkOrphanedMethods(filePath string, routedMethods map[string]bool) []types.Issue {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return nil
	}

	relPath := shortenPath(filePath)
	var issues []types.Issue

	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || !fn.Name.IsExported() {
			return true
		}

		for _, field := range fn.Recv.List {
			typeName := extractReceiverTypeName(field.Type)
			if !strings.HasSuffix(typeName, "Handler") {
				continue
			}

			methodName := fn.Name.Name
			// Skip infrastructure methods that are never route handlers.
			if methodName == "RegisterRoutes" || strings.HasPrefix(methodName, "New") {
				continue
			}

			if !routedMethods[methodName] {
				pos := fset.Position(fn.Pos())
				issues = append(issues, types.Issue{
					Kind:     types.KindDeadRoute,
					Severity: types.SeverityLow,
					File:     relPath,
					Line:     pos.Line,
					Message: fmt.Sprintf("handler method %s.%s is declared but not registered in any route",
						typeName, methodName),
				})
			}
		}
		return true
	})

	return issues
}

// collectDeclaredHandlerMethods returns all exported method names on *Handler types.
func collectDeclaredHandlerMethods(handlerDir string) map[string]bool {
	declared := make(map[string]bool)

	filepath.WalkDir(handlerDir, func(path string, d fs.DirEntry, err error) error { //nolint:errcheck
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || isTestFile(path) {
			return nil
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil
		}
		ast.Inspect(f, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || !fn.Name.IsExported() {
				return true
			}
			for _, field := range fn.Recv.List {
				if strings.HasSuffix(extractReceiverTypeName(field.Type), "Handler") {
					declared[fn.Name.Name] = true
				}
			}
			return true
		})
		return nil
	})

	return declared
}

// findRouterFiles returns all router/app Go files in the given directory tree.
func findRouterFiles(dir string) []string {
	var files []string
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error { //nolint:errcheck
		if err != nil || d.IsDir() {
			return nil
		}
		base := strings.ToLower(d.Name())
		if base == "router.go" || base == "app.go" || strings.HasSuffix(base, "_router.go") {
			files = append(files, path)
		}
		return nil
	})
	return files
}

// extractReceiverTypeName extracts the type name from a receiver field type expression.
// Handles both *T and T forms.
func extractReceiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return extractReceiverTypeName(t.X)
	case *ast.Ident:
		return t.Name
	}
	return ""
}
