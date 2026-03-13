// Package parser provides Go AST utilities for reading existing project code.
//
// This package enables go-tk to analyze target projects WITHOUT executing them:
//   - Extract struct definitions (for CRUD field inference)
//   - Discover HTTP routes (for test generation and dead route analysis)
//   - Parse imports (for circular dependency detection)
//
// Architecture Decision:
// We use go/ast (stdlib) instead of reflection because:
//  1. Static analysis — no need to compile target project
//  2. Works on broken code (useful during development)
//  3. Zero runtime overhead (parse at tool invocation, not in generated app)
//
// Thread safety: All functions are stateless and safe for concurrent use.
//
// Error handling: Graceful degradation — if we can't parse a file, we skip it
// rather than fail the entire command. This makes go-tk resilient to syntax
// errors in the target project.
package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// ParsedStruct holds the extracted definition of a Go struct.
//
// Used by CRUD generator to infer field types from existing entities
// (future feature: "go-tk gen crud User --from-entity").
type ParsedStruct struct {
	Name   string        // Struct name (e.g. "User")
	Fields []ParsedField // All fields in declaration order
}

// ParsedField holds information about a single struct field.
//
// Includes metadata beyond what's available at runtime via reflection:
//   - Struct tags (for DB column mapping, JSON serialization)
//   - Comments (for OpenAPI documentation generation)
type ParsedField struct {
	Name    string            // Field name (e.g. "Email")
	Type    string            // Go type as string (e.g. "string", "time.Time", "*int")
	Tags    map[string]string // Struct tags (json, db, validate, etc.)
	Comment string            // Inline or doc comment
}

// ParseStructFromFile opens a Go source file and extracts the named struct.
// Returns ErrStructNotFound if the struct does not exist in the file.
//
// Algorithm:
//  1. Parse file to AST (fails on syntax errors)
//  2. Walk all top-level declarations
//  3. Find type declaration matching structName
//  4. Extract fields with their types, tags, and comments
//
// Limitations:
//   - Only parses top-level struct declarations (not embedded in functions)
//   - Anonymous fields (embedding) have empty Name
//   - Complex types (chan, func) are represented as "interface{}"
//
// Use case:
//
//	Infer CRUD fields from existing entity definition rather than CLI flags.
//
// Example:
//
//	parsed, err := ParseStructFromFile("internal/domain/entity/user.go", "User")
//	for _, field := range parsed.Fields {
//	    fmt.Printf("%s %s `%s`\n", field.Name, field.Type, field.Tags["json"])
//	}
func ParseStructFromFile(filePath, structName string) (*ParsedStruct, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", filePath, err)
	}

	// Walk all top-level declarations
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue // Not a type declaration
		}

		// Check all type specs in this declaration (may have multiple in one block)
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != structName {
				continue // Not our target struct
			}

			// Ensure it's actually a struct (not an interface or type alias)
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				return nil, fmt.Errorf("%s is not a struct", structName)
			}

			// Extract all fields
			ps := &ParsedStruct{Name: structName}
			for _, field := range structType.Fields.List {
				pf := ParsedField{
					Type: typeToString(field.Type),
					Tags: parseTags(field.Tag),
				}
				if field.Comment != nil {
					pf.Comment = strings.TrimSpace(field.Comment.Text())
				}
				// field.Names can be empty for anonymous fields (embedding)
				if len(field.Names) > 0 {
					pf.Name = field.Names[0].Name
				}
				// Only include named fields (skip embedded types for now)
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
//
// Handles common type expressions:
//
//	*ast.Ident        → "string", "int", "MyType"
//	*ast.SelectorExpr → "time.Time", "uuid.UUID"
//	*ast.StarExpr     → "*User", "*int"
//	*ast.ArrayType    → "[]string", "[...]int"
//	*ast.MapType      → "map[string]int"
//
// Fallback for unsupported types (chan, func, struct literal) → "interface{}"
//
// This is a best-effort conversion — complex nested types may not be perfect.
// Good enough for CRUD generation where most fields are primitives or common types.
func typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		// pkg.Type (e.g. time.Time)
		return typeToString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		// Pointer type
		return "*" + typeToString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			// Slice: []T
			return "[]" + typeToString(t.Elt)
		}
		// Array: [N]T → represent as [...]T
		return "[...]" + typeToString(t.Elt)
	case *ast.MapType:
		return "map[" + typeToString(t.Key) + "]" + typeToString(t.Value)
	default:
		// Unsupported: chan, func, struct literal, interface
		return "interface{}"
	}
}

// parseTags parses a struct tag string into a key→value map.
//
// Struct tag format: `json:"name,omitempty" db:"user_name" validate:"required"`
// Returns: {"json": "name,omitempty", "db": "user_name", "validate": "required"}
//
// Limitations:
//   - Does not handle quoted values with spaces (rare in practice)
//   - Assumes tags are well-formed (no validation)
//
// This is sufficient for extracting tag keys for CRUD generation.
// For production tag parsing, use reflect.StructTag.
func parseTags(lit *ast.BasicLit) map[string]string {
	tags := make(map[string]string)
	if lit == nil {
		return tags
	}

	// Raw tag string is backtick-delimited: `json:"..." db:"..."`
	raw := strings.Trim(lit.Value, "`")

	// Split by whitespace (tags are space-separated)
	for _, part := range strings.Fields(raw) {
		// Each part: key:"value"
		kv := strings.SplitN(part, ":", 2)
		if len(kv) == 2 {
			// Strip quotes from value
			tags[kv[0]] = strings.Trim(kv[1], `"`)
		}
	}
	return tags
}
