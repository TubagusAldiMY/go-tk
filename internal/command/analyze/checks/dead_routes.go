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
			addRoutedMethod(r.Handler, routedMethods)
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
	issues = append(issues, checkDeadRouteRegistrations(routerFiles, declaredMethods)...)

	return issues, scanned, nil
}

// checkDeadRouteRegistrations finds routes whose handler method is not declared in any handler file.
func checkDeadRouteRegistrations(routerFiles []string, declaredMethods map[string]bool) []types.Issue {
	var issues []types.Issue
	for _, rf := range routerFiles {
		routes, err := goparser.ParseRoutesFromFile(rf)
		if err != nil {
			continue
		}
		relPath := shortenPath(rf)
		for _, r := range routes {
			if issue := deadRouteIssue(r.Handler, r.Method, r.Path, relPath, declaredMethods); issue != nil {
				issues = append(issues, *issue)
			}
		}
	}
	return issues
}

// addRoutedMethod extracts the method name from a handler string ("var.Method") and records it.
func addRoutedMethod(handler string, routedMethods map[string]bool) {
	idx := strings.LastIndex(handler, ".")
	if idx < 0 {
		return
	}
	if m := handler[idx+1:]; m != "" && m != "<unknown>" {
		routedMethods[m] = true
	}
}

// deadRouteIssue returns an Issue if the handler method is not in declaredMethods, nil otherwise.
func deadRouteIssue(handler, method, path, relPath string, declared map[string]bool) *types.Issue {
	idx := strings.LastIndex(handler, ".")
	if idx < 0 {
		return nil
	}
	m := handler[idx+1:]
	if m == "" || m == "<unknown>" || m == "func" {
		return nil
	}
	if declared[m] {
		return nil
	}
	issue := types.Issue{
		Kind:     types.KindDeadRoute,
		Severity: types.SeverityHigh,
		File:     relPath,
		Message:  fmt.Sprintf("route %s %s references handler method %q which is not declared in any handler file", method, path, m),
	}
	return &issue
}

// orphanedMethodIssue checks a single receiver field and returns an issue if the method is unrouted.
func orphanedMethodIssue(field *ast.Field, fn *ast.FuncDecl, fset *token.FileSet, relPath string, routedMethods map[string]bool) *types.Issue {
	typeName := extractReceiverTypeName(field.Type)
	if !strings.HasSuffix(typeName, "Handler") {
		return nil
	}
	methodName := fn.Name.Name
	if methodName == "RegisterRoutes" || strings.HasPrefix(methodName, "New") {
		return nil
	}
	if routedMethods[methodName] {
		return nil
	}
	pos := fset.Position(fn.Pos())
	issue := types.Issue{
		Kind:     types.KindDeadRoute,
		Severity: types.SeverityLow,
		File:     relPath,
		Line:     pos.Line,
		Message:  fmt.Sprintf("handler method %s.%s is declared but not registered in any route", typeName, methodName),
	}
	return &issue
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
			if issue := orphanedMethodIssue(field, fn, fset, relPath, routedMethods); issue != nil {
				issues = append(issues, *issue)
			}
		}
		return true
	})

	return issues
}

// recordHandlerMethods parses a single Go file and records exported handler method names.
func recordHandlerMethods(path string, declared map[string]bool) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return
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
}

// collectDeclaredHandlerMethods returns all exported method names on *Handler types.
func collectDeclaredHandlerMethods(handlerDir string) map[string]bool {
	declared := make(map[string]bool)
	filepath.WalkDir(handlerDir, func(path string, d fs.DirEntry, err error) error { //nolint:errcheck
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || isTestFile(path) {
			return nil
		}
		recordHandlerMethods(path, declared)
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
