// Package checks contains individual analysis checks for go-tk types.
package checks

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/TubagusAldiMY/go-tk/internal/command/analyze/types"
)

// CheckUnhandledErrors scans Go source files in dir for errors assigned to _.
// Pattern: `_, err := f()` followed by no check, or `_ = f()` where f returns error.
func CheckUnhandledErrors(dir string) ([]types.Issue, int, error) {
	var issues []types.Issue
	scanned := 0

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || isTestFile(path) {
			return nil
		}
		scanned++
		fileIssues := checkFileForUnhandledErrors(path)
		issues = append(issues, fileIssues...)
		return nil
	})

	return issues, scanned, err
}

func checkFileForUnhandledErrors(filePath string) []types.Issue {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, src, 0)
	if err != nil {
		return nil
	}

	relPath := shortenPath(filePath)
	var issues []types.Issue

	ast.Inspect(f, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}

		// Look for patterns like: _ = someFunc() or _, err = ...; _ = err
		for _, lhs := range assign.Lhs {
			ident, ok := lhs.(*ast.Ident)
			if !ok || ident.Name != "_" {
				continue
			}

			// Check if RHS is a function call returning an error
			for _, rhs := range assign.Rhs {
				call, ok := rhs.(*ast.CallExpr)
				if !ok {
					continue
				}
				// Heuristic: if the function name ends in a verb (Close, Remove, Write...)
				// and we're discarding the result, flag it.
				if funcName := callFuncName(call); isErrorReturningFunc(funcName) {
					pos := fset.Position(assign.Pos())
					issues = append(issues, types.Issue{
						Kind:     types.KindUnhandledError,
						Severity: types.SeverityMedium,
						File:     relPath,
						Line:     pos.Line,
						Message:  "error return value discarded from " + funcName + "()",
					})
				}
			}
		}
		return true
	})

	return issues
}

// callFuncName extracts the function name from a call expression.
func callFuncName(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name
	case *ast.SelectorExpr:
		return fn.Sel.Name
	}
	return ""
}

// isErrorReturningFunc is a heuristic: common functions whose errors are often ignored.
var errorReturningFuncs = map[string]bool{
	"Close": true, "Write": true, "Flush": true, "Remove": true,
	"Rename": true, "Mkdir": true, "MkdirAll": true, "WriteFile": true,
	"Sync": true, "Truncate": true, "Chmod": true, "Chown": true,
}

func isErrorReturningFunc(name string) bool {
	return errorReturningFuncs[name]
}
