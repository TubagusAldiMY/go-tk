package checks

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/TubagusAldiMY/go-tk/internal/command/analyze/types"
)

// reLocalhost matches hardcoded local host addresses.
var reLocalhost = regexp.MustCompile(`\b(localhost|127\.0\.0\.1)\b`)

var suspiciousKeywords = []string{
	"password", "secret", "token", "api_key", "apikey",
	"private_key", "access_key",
}

// CheckHardcodedValues finds string literals that look like credentials or config.
func CheckHardcodedValues(dir string) ([]types.Issue, int, error) {
	var issues []types.Issue
	scanned := 0

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || isTestFile(path) {
			return nil
		}
		scanned++
		issues = append(issues, checkHardcoded(path)...)
		return nil
	})

	return issues, scanned, err
}

func checkHardcoded(filePath string) []types.Issue {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return nil
	}

	relPath := shortenPath(filePath)
	var issues []types.Issue

	ast.Inspect(f, func(n ast.Node) bool {
		// Look for assignments: varName = "suspicious-value"
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}

		for i, lhs := range assign.Lhs {
			ident, ok := lhs.(*ast.Ident)
			if !ok {
				continue
			}
			varName := strings.ToLower(ident.Name)

			if i >= len(assign.Rhs) {
				continue
			}
			lit, ok := assign.Rhs[i].(*ast.BasicLit)
			if !ok || lit.Kind.String() != "STRING" {
				continue
			}
			val := strings.Trim(lit.Value, `"`)

			// Check variable name for sensitive keywords
			for _, kw := range suspiciousKeywords {
				if strings.Contains(varName, kw) && len(val) > 4 {
					pos := fset.Position(assign.Pos())
					issues = append(issues, types.Issue{
						Kind:     types.KindHardcodedValue,
						Severity: types.SeverityCritical,
						File:     relPath,
						Line:     pos.Line,
						Message:  "possible hardcoded credential in variable '" + ident.Name + "'",
					})
					break
				}
			}

			// Flag hardcoded localhost in non-config files
			if reLocalhost.MatchString(val) && !isConfigFile(filePath) {
				pos := fset.Position(assign.Pos())
				issues = append(issues, types.Issue{
					Kind:     types.KindHardcodedValue,
					Severity: types.SeverityMedium,
					File:     relPath,
					Line:     pos.Line,
					Message:  "hardcoded host address '" + val + "' should come from config/env",
				})
			}
		}
		return true
	})

	return issues
}

func isConfigFile(path string) bool {
	return strings.Contains(path, "config") || strings.Contains(path, "test")
}
