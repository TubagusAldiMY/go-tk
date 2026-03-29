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

// credentialIssue returns an Issue if the assignment looks like a hardcoded credential, nil otherwise.
func credentialIssue(fset *token.FileSet, assign *ast.AssignStmt, ident *ast.Ident, val, relPath string) *types.Issue {
	varName := strings.ToLower(ident.Name)
	for _, kw := range suspiciousKeywords {
		if strings.Contains(varName, kw) && len(val) > 4 {
			pos := fset.Position(assign.Pos())
			issue := types.Issue{
				Kind:     types.KindHardcodedValue,
				Severity: types.SeverityCritical,
				File:     relPath,
				Line:     pos.Line,
				Message:  "possible hardcoded credential in variable '" + ident.Name + "'",
			}
			return &issue
		}
	}
	return nil
}

// localhostIssue returns an Issue if the value is a hardcoded localhost address, nil otherwise.
func localhostIssue(fset *token.FileSet, assign *ast.AssignStmt, val, filePath, relPath string) *types.Issue {
	if !reLocalhost.MatchString(val) || isConfigFile(filePath) {
		return nil
	}
	pos := fset.Position(assign.Pos())
	issue := types.Issue{
		Kind:     types.KindHardcodedValue,
		Severity: types.SeverityMedium,
		File:     relPath,
		Line:     pos.Line,
		Message:  "hardcoded host address '" + val + "' should come from config/env",
	}
	return &issue
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
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		issues = append(issues, checkAssignForHardcoded(fset, assign, filePath, relPath)...)
		return true
	})

	return issues
}

// checkAssignForHardcoded inspects each LHS/RHS pair in an assignment for hardcoded values.
func checkAssignForHardcoded(fset *token.FileSet, assign *ast.AssignStmt, filePath, relPath string) []types.Issue {
	var issues []types.Issue
	for i, lhs := range assign.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok || i >= len(assign.Rhs) {
			continue
		}
		lit, ok := assign.Rhs[i].(*ast.BasicLit)
		if !ok || lit.Kind.String() != "STRING" {
			continue
		}
		val := strings.Trim(lit.Value, `"`)
		if issue := credentialIssue(fset, assign, ident, val, relPath); issue != nil {
			issues = append(issues, *issue)
		}
		if issue := localhostIssue(fset, assign, val, filePath, relPath); issue != nil {
			issues = append(issues, *issue)
		}
	}
	return issues
}

func isConfigFile(path string) bool {
	return strings.Contains(path, "config") || strings.Contains(path, "test")
}
