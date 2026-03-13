// Package parser provides Go AST utilities for analyzing target projects.
//
// This file implements HTTP route discovery for Gin and Fiber frameworks.
// Used by:
//   - "go-tk test" to generate smoke tests for all endpoints
//   - "go-tk analyze" to detect dead routes and orphaned handlers
//
// Supported patterns:
//
//	Gin:   router.GET("/path", handler.Method)
//	Fiber: app.Post("/path", handler.Method)
//
// Limitations:
//   - Only detects direct route registrations (not route groups with closures)
//   - Dynamic paths from variables are not captured
//   - Middleware is not tracked (analyzed separately by CheckMissingAuth)
//
// This is best-effort parsing — if we can't parse a route, we skip it.
// Better to miss a route than fail the entire command on unusual patterns.
package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// Route holds an extracted HTTP route registration.
//
// Example Gin route:
//
//	router.GET("/api/v1/users/:id", userHandler.GetByID)
//	→ {Method: "GET", Path: "/api/v1/users/:id", Handler: "userHandler.GetByID"}
//
// Example Fiber route:
//
//	app.Post("/api/v1/products", productHandler.Create)
//	→ {Method: "POST", Path: "/api/v1/products", Handler: "productHandler.Create"}
type Route struct {
	Method  string // HTTP method: GET, POST, PUT, DELETE, PATCH
	Path    string // Route path (may include params: /users/:id)
	Handler string // Handler identifier (receiver.Method or function name)
}

// knownHTTPMethods is the set of gin/fiber method names we look for.
//
// Both frameworks use the same method names (GET, POST, etc.) as function names.
// We normalize to uppercase for consistency.
var knownHTTPMethods = map[string]string{
	"GET":    "GET",
	"POST":   "POST",
	"PUT":    "PUT",
	"DELETE": "DELETE",
	"PATCH":  "PATCH",
}

// ParseRoutesFromFile reads a Go source file and extracts all route
// registrations of the form receiver.METHOD("path", handler).
//
// Algorithm:
//  1. Parse file to AST
//  2. Walk all nodes looking for CallExpr (function calls)
//  3. Check if call is a SelectorExpr (receiver.Method)
//  4. Check if method name is in knownHTTPMethods (GET, POST, etc.)
//  5. Extract first arg as path (string literal)
//  6. Extract last arg as handler (function reference)
//
// Graceful degradation:
//   - If a route call doesn't match expected pattern, skip it
//   - If path is not a string literal (e.g. variable), skip it
//   - If handler is complex expression, represent as "<unknown>"
//
// This is best-effort — unusual patterns may be missed, but we never crash.
//
// Example routes detected:
//
//	router.GET("/users", handler.List)        → ✅
//	app.Post("/products/:id", h.Update)       → ✅
//	r.PUT(pathVar, handler)                   → ❌ skipped (path not literal)
//	group.GET("/admin", middleware, handler)  → ⚠️ detected (middleware ignored)
//
// Returns: All successfully parsed routes, or empty slice if file unparseable.
func ParseRoutesFromFile(filePath string) ([]Route, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", filePath, err)
	}

	var routes []Route

	// Walk AST looking for route registration calls
	ast.Inspect(f, func(n ast.Node) bool {
		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true // Not a function call, continue walking
		}

		// Check if call is receiver.Method (e.g. router.GET)
		sel, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true // Not a selector expression, skip
		}

		// Check if method name is an HTTP method (GET, POST, etc.)
		method := strings.ToUpper(sel.Sel.Name)
		if _, exists := knownHTTPMethods[method]; !exists {
			return true // Not an HTTP method, skip
		}

		// Expect at least 2 args: (path, handler) or (path, middleware..., handler)
		if len(callExpr.Args) < 2 {
			return true // Invalid route signature, skip
		}

		// Extract path (first argument, must be string literal)
		pathLit, ok := callExpr.Args[0].(*ast.BasicLit)
		if !ok {
			return true // Path is not a literal (e.g. variable), skip
		}
		routePath := strings.Trim(pathLit.Value, `"`)

		// Extract handler (last argument, may be function or receiver.Method)
		handlerStr := handlerExprToString(callExpr.Args[len(callExpr.Args)-1])

		routes = append(routes, Route{
			Method:  method,
			Path:    routePath,
			Handler: handlerStr,
		})

		return true // Continue walking
	})

	return routes, nil
}

// handlerExprToString converts a handler expression to a string identifier.
//
// Handles:
//
//	handler.Create      → "handler.Create" (receiver.Method)
//	CreateUser          → "CreateUser" (function name)
//	pkg.Handler.Method  → "pkg.Handler.Method" (fully qualified)
//
// Returns "<unknown>" for complex expressions (e.g. inline closures, method values).
//
// This is for display/matching purposes — we don't need perfect precision.
func handlerExprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		// receiver.Method
		return exprToString(e.X) + "." + e.Sel.Name
	case *ast.Ident:
		// Function name
		return e.Name
	default:
		// Closure, method value, etc.
		return "<unknown>"
	}
}

// exprToString recursively converts an expression to a dotted identifier.
//
// Used by handlerExprToString to handle nested selectors:
//
//	pkg.Type.Method → "pkg.Type.Method"
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
