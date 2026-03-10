package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// Route holds an extracted HTTP route registration.
type Route struct {
	Method  string // GET, POST, PUT, DELETE, PATCH
	Path    string // e.g. "/api/v1/users/:id"
	Handler string // e.g. "userHandler.GetByID"
}

// knownHTTPMethods is the set of gin/fiber method names we look for.
var knownHTTPMethods = map[string]string{
	"GET":    "GET",
	"POST":   "POST",
	"PUT":    "PUT",
	"DELETE": "DELETE",
	"PATCH":  "PATCH",
}

// ParseRoutesFromFile reads a Go source file and extracts all route
// registrations of the form receiver.METHOD("path", handler).
// It is best-effort: if it cannot parse a call it skips it gracefully.
func ParseRoutesFromFile(filePath string) ([]Route, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", filePath, err)
	}

	var routes []Route

	ast.Inspect(f, func(n ast.Node) bool {
		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		method := strings.ToUpper(sel.Sel.Name)
		if _, exists := knownHTTPMethods[method]; !exists {
			return true
		}

		if len(callExpr.Args) < 2 {
			return true
		}

		pathLit, ok := callExpr.Args[0].(*ast.BasicLit)
		if !ok {
			return true
		}
		routePath := strings.Trim(pathLit.Value, `"`)

		handlerStr := handlerExprToString(callExpr.Args[len(callExpr.Args)-1])

		routes = append(routes, Route{
			Method:  method,
			Path:    routePath,
			Handler: handlerStr,
		})

		return true
	})

	return routes, nil
}

func handlerExprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		return exprToString(e.X) + "." + e.Sel.Name
	case *ast.Ident:
		return e.Name
	default:
		return "<unknown>"
	}
}

func exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return exprToString(e.X) + "." + e.Sel.Name
	default:
		return ""
	}
}
