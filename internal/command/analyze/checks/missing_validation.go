package checks

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/TubagusAldiMY/go-tk/internal/command/analyze/types"
)

// CheckMissingValidation finds Gin handler functions that bind JSON
// but do not call validator.Struct or Validate.
func CheckMissingValidation(handlersDir string) ([]types.Issue, int, error) {
	var issues []types.Issue
	scanned := 0

	err := filepath.WalkDir(handlersDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || isTestFile(path) {
			return nil
		}
		scanned++
		issues = append(issues, checkHandlerValidation(path)...)
		return nil
	})

	return issues, scanned, err
}

func checkHandlerValidation(filePath string) []types.Issue {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return nil
	}

	relPath := shortenPath(filePath)
	var issues []types.Issue

	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return true
		}

		hasBind := false
		hasValidate := false

		ast.Inspect(fn.Body, func(inner ast.Node) bool {
			call, ok := inner.(*ast.CallExpr)
			if !ok {
				return true
			}
			name := selectorName(call)
			if strings.Contains(name, "ShouldBindJSON") || strings.Contains(name, "BindJSON") {
				hasBind = true
			}
			if strings.Contains(name, "Struct") || strings.Contains(name, "Validate") {
				hasValidate = true
			}
			return true
		})

		if hasBind && !hasValidate {
			pos := fset.Position(fn.Pos())
			issues = append(issues, types.Issue{
				Kind:     types.KindMissingValidation,
				Severity: types.SeverityHigh,
				File:     relPath,
				Line:     pos.Line,
				Message:  "handler " + fn.Name.Name + "() binds JSON but does not validate input",
			})
		}
		return true
	})

	return issues
}

// selectorName returns "X.Sel" for a selector call, or just the name for ident calls.
func selectorName(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.SelectorExpr:
		return fn.Sel.Name
	case *ast.Ident:
		return fn.Name
	}
	return ""
}
