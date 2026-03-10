package checks

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"github.com/TubagusAldiMY/go-tk/internal/command/analyze/types"
)

// mutableMethods are HTTP methods that mutate state and typically require authentication.
var mutableMethods = map[string]bool{
	"POST": true, "PUT": true, "PATCH": true, "DELETE": true,
	// Fiber/Gin capitalize differently — both handled via ToUpper in parser
	"Post": true, "Put": true, "Patch": true, "Delete": true,
}

// sensitivePaths are path segments that imply privileged operations.
var sensitivePaths = []string{
	"user", "admin", "account", "profile", "password",
	"token", "payment", "order", "billing", "role", "permission",
}

// CheckMissingAuth heuristically detects routes that use mutable HTTP methods
// (POST/PUT/PATCH/DELETE) on sensitive paths without visible auth middleware.
//
// This is a HEURISTIC check — it cannot guarantee completeness.
// It flags routes as LOW severity to prompt manual review.
func CheckMissingAuth(internalDir string) ([]types.Issue, int, error) {
	scanned := 0
	var issues []types.Issue

	for _, rf := range findRouterFiles(internalDir) {
		scanned++
		fileIssues := checkRouterFileForMissingAuth(rf)
		issues = append(issues, fileIssues...)
	}

	return issues, scanned, nil
}

func checkRouterFileForMissingAuth(filePath string) []types.Issue {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return nil
	}

	relPath := shortenPath(filePath)
	var issues []types.Issue

	// Inspect every function in the router file.
	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return true
		}

		// Does this function have any auth/jwt middleware call at all?
		hasAuthMiddleware := functionHasAuthMiddleware(fn.Body)

		// Collect all route registrations with mutable methods.
		ast.Inspect(fn.Body, func(inner ast.Node) bool {
			call, ok := inner.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			method := sel.Sel.Name
			if !mutableMethods[method] {
				return true
			}

			if len(call.Args) < 2 {
				return true
			}

			pathLit, ok := call.Args[0].(*ast.BasicLit)
			if !ok {
				return true
			}
			routePath := strings.Trim(pathLit.Value, `"`)

			// Only flag if the path looks sensitive.
			if !isSensitivePath(routePath) {
				return true
			}

			if !hasAuthMiddleware {
				pos := fset.Position(call.Pos())
				issues = append(issues, types.Issue{
					Kind:     types.KindMissingAuth,
					Severity: types.SeverityLow,
					File:     relPath,
					Line:     pos.Line,
					Message: fmt.Sprintf(
						"route %s %q may be missing auth middleware — no auth/jwt middleware found in router function (heuristic)",
						strings.ToUpper(method), routePath,
					),
				})
			}

			return true
		})

		return true
	})

	return issues
}

// functionHasAuthMiddleware returns true if the function body contains a call
// whose name or arguments reference "auth", "jwt", or "Auth"/"JWT" (case-sensitive).
func functionHasAuthMiddleware(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		callStr := exprCallString(call)
		lower := strings.ToLower(callStr)
		if strings.Contains(lower, "auth") || strings.Contains(lower, "jwt") {
			found = true
			return false
		}
		return true
	})
	return found
}

// exprCallString converts a call expression to a representative string for matching.
func exprCallString(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.SelectorExpr:
		return selectorCallStr(fn)
	case *ast.Ident:
		return fn.Name
	}
	return ""
}

func selectorCallStr(sel *ast.SelectorExpr) string {
	switch x := sel.X.(type) {
	case *ast.Ident:
		return x.Name + "." + sel.Sel.Name
	case *ast.SelectorExpr:
		return selectorCallStr(x) + "." + sel.Sel.Name
	}
	return sel.Sel.Name
}

// isSensitivePath returns true if the path contains a sensitive segment keyword.
func isSensitivePath(path string) bool {
	lower := strings.ToLower(path)
	for _, kw := range sensitivePaths {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}
