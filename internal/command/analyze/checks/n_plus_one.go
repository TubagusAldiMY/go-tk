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

// CheckNPlusOne scans for database calls inside loop bodies (heuristic N+1 detection).
func CheckNPlusOne(dir string) ([]types.Issue, int, error) {
	var issues []types.Issue
	scanned := 0

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || isTestFile(path) {
			return nil
		}
		scanned++
		issues = append(issues, checkNPlusOne(path)...)
		return nil
	})

	return issues, scanned, err
}

// dbMethodNames are GORM methods that indicate a database query.
var dbMethodNames = map[string]bool{
	"Find": true, "First": true, "Last": true, "Take": true,
	"Create": true, "Save": true, "Update": true, "Updates": true,
	"Delete": true, "Count": true, "Raw": true, "Exec": true,
	"Preload": true, "Joins": true,
}

func checkNPlusOne(filePath string) []types.Issue {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return nil
	}

	relPath := shortenPath(filePath)
	var issues []types.Issue

	ast.Inspect(f, func(n ast.Node) bool {
		// Look for for/range loops
		var loopBody *ast.BlockStmt
		var loopPos token.Pos

		switch loop := n.(type) {
		case *ast.ForStmt:
			loopBody = loop.Body
			loopPos = loop.For
		case *ast.RangeStmt:
			loopBody = loop.Body
			loopPos = loop.For
		default:
			return true
		}

		// Scan loop body for DB calls
		ast.Inspect(loopBody, func(inner ast.Node) bool {
			call, ok := inner.(*ast.CallExpr)
			if !ok {
				return true
			}
			if name := selectorName(call); dbMethodNames[name] {
				pos := fset.Position(loopPos)
				issues = append(issues, types.Issue{
					Kind:     types.KindNPlusOne,
					Severity: types.SeverityHigh,
					File:     relPath,
					Line:     pos.Line,
					Message:  "potential N+1 query: DB call '" + name + "' inside loop",
				})
				return false // don't recurse further into this loop
			}
			return true
		})
		return true
	})

	return issues
}
