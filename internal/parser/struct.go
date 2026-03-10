// Package parser provides Go AST utilities for reading existing project code.
package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// ParsedStruct holds the extracted definition of a Go struct.
type ParsedStruct struct {
	Name   string
	Fields []ParsedField
}

// ParsedField holds information about a single struct field.
type ParsedField struct {
	Name    string
	Type    string
	Tags    map[string]string
	Comment string
}

// ParseStructFromFile opens a Go source file and extracts the named struct.
// Returns ErrStructNotFound if the struct does not exist in the file.
func ParseStructFromFile(filePath, structName string) (*ParsedStruct, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", filePath, err)
	}

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != structName {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				return nil, fmt.Errorf("%s is not a struct", structName)
			}

			ps := &ParsedStruct{Name: structName}
			for _, field := range structType.Fields.List {
				pf := ParsedField{
					Type: typeToString(field.Type),
					Tags: parseTags(field.Tag),
				}
				if field.Comment != nil {
					pf.Comment = strings.TrimSpace(field.Comment.Text())
				}
				if len(field.Names) > 0 {
					pf.Name = field.Names[0].Name
				}
				if pf.Name != "" {
					ps.Fields = append(ps.Fields, pf)
				}
			}
			return ps, nil
		}
	}

	return nil, fmt.Errorf("%w: %s in %s", ErrStructNotFound, structName, filePath)
}

// typeToString converts an ast.Expr to a type string representation.
func typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return typeToString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + typeToString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + typeToString(t.Elt)
		}
		return "[...]" + typeToString(t.Elt)
	case *ast.MapType:
		return "map[" + typeToString(t.Key) + "]" + typeToString(t.Value)
	default:
		return "interface{}"
	}
}

// parseTags parses a struct tag string into a key→value map.
func parseTags(lit *ast.BasicLit) map[string]string {
	tags := make(map[string]string)
	if lit == nil {
		return tags
	}

	raw := strings.Trim(lit.Value, "`")
	for _, part := range strings.Fields(raw) {
		kv := strings.SplitN(part, ":", 2)
		if len(kv) == 2 {
			tags[kv[0]] = strings.Trim(kv[1], `"`)
		}
	}
	return tags
}
