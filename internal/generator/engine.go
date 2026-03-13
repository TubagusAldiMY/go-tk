// Package generator provides template rendering and file generation utilities.
//
// The Engine type wraps Go's text/template with embedded filesystem support,
// enabling compile-time template bundling via go:embed. This package is the core
// of all code generation in go-tk (project scaffolding, CRUD generation, etc.).
//
// Architecture Decision (ADR-001):
// We use text/template (stdlib) instead of external engines (Mustache, Pongo2)
// to minimize dependencies and leverage Go's native template safety features.
// Complex logic stays in Go code; templates remain presentation-only.
//
// Key features:
//   - Atomic file writes (temp + rename) to prevent partial writes
//   - Auto-formatting for .go files (gofmt + goimports)
//   - Custom template functions (toSnake, toPascal, etc.) via FuncMap
//   - Directory traversal for bulk template discovery
package generator

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"text/template"
)

// Engine renders text/template templates from an embedded filesystem.
//
// Usage pattern:
//  1. Embed templates in templates/ directory via go:embed
//  2. Create Engine with NewEngine(fsys, "templates")
//  3. Call Render(templatePath, data) to execute templates
//  4. Use WriteAtomic() to safely write output to disk
//
// Thread safety: Engine is safe for concurrent reads after construction.
// Each Render() call parses the template independently (stateless).
type Engine struct {
	fs      embed.FS         // Embedded filesystem containing template files
	baseDir string           // Root directory within fs (e.g. "templates")
	funcs   template.FuncMap // Custom template functions (toSnake, toPascal, etc.)
}

// NewEngine creates a new template Engine backed by the given embed.FS.
//
// Parameters:
//
//	fsys    — Embedded filesystem (from go:embed directive)
//	baseDir — Root directory within fsys that contains templates (e.g. "templates")
//
// Example:
//
//	//go:embed all:templates
//	var templateFS embed.FS
//	engine := generator.NewEngine(templateFS, "templates")
//
// The engine is immediately ready to use and does not require additional setup.
// Custom template functions (toSnake, toPascal, pluralize) are auto-registered.
func NewEngine(fsys embed.FS, baseDir string) *Engine {
	return &Engine{
		fs:      fsys,
		baseDir: baseDir,
		funcs:   defaultFuncMap(),
	}
}

// Render parses the template at templatePath (relative to baseDir) and
// executes it with data, returning the rendered bytes.
//
// Parameters:
//
//	templatePath — Path relative to baseDir (e.g. "crud/entity.go.tmpl")
//	data         — Template context data (any Go type; usually a struct)
//
// Returns:
//
//	Rendered template as []byte, or error if template not found/invalid.
//
// Example:
//
//	type TemplateData struct { Name string; Fields []Field }
//	output, err := engine.Render("crud/entity.go.tmpl", TemplateData{...})
//
// Error handling:
//   - Template not found → fs.ErrNotExist wrapped
//   - Parse error → syntax error with line number
//   - Execution error → runtime error (e.g. nil pointer in template)
//
// This method is stateless — safe to call concurrently.
func (e *Engine) Render(templatePath string, data any) ([]byte, error) {
	fullPath := e.baseDir + "/" + templatePath

	// Read template source from embedded FS
	src, err := fs.ReadFile(e.fs, fullPath)
	if err != nil {
		return nil, fmt.Errorf("reading template %s: %w", fullPath, err)
	}

	// Parse template with custom functions (toSnake, toPascal, etc.)
	tmpl, err := template.New(templatePath).Funcs(e.funcs).Parse(string(src))
	if err != nil {
		return nil, fmt.Errorf("parsing template %s: %w", fullPath, err)
	}

	// Execute template — writes output to buffer
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template %s: %w", fullPath, err)
	}

	return buf.Bytes(), nil
}

// RenderString renders a raw template string (not from the FS) with data.
//
// Useful for inline templates or dynamically constructed template content
// (e.g. gotk.yaml config file, one-off snippets).
//
// Parameters:
//
//	name    — Template name (for error messages; does not affect output)
//	tmplStr — Raw template content as string
//	data    — Template context data
//
// Example:
//
//	output, err := engine.RenderString("config", "version: {{.Version}}", data)
//
// This is identical to Render() but bypasses the embedded filesystem.
// Use Render() for pre-packaged templates, RenderString() for dynamic content.
func (e *Engine) RenderString(name, tmplStr string, data any) ([]byte, error) {
	tmpl, err := template.New(name).Funcs(e.funcs).Parse(tmplStr)
	if err != nil {
		return nil, fmt.Errorf("parsing inline template %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing inline template %s: %w", name, err)
	}

	return buf.Bytes(), nil
}

// ListTemplates returns all template paths under a subdirectory of baseDir.
//
// This is used for bulk operations (e.g. "generate all files in project/gin-postgres/").
//
// Parameters:
//
//	subDir — Subdirectory relative to baseDir (e.g. "project/gin-postgres").
//	         Empty string "" lists all templates in baseDir.
//
// Returns:
//
//	[]string — Paths relative to baseDir (e.g. ["project/gin-postgres/main.go.tmpl", ...])
//
// Example:
//
//	paths, _ := engine.ListTemplates("project/gin-postgres")
//	for _, p := range paths {
//	    content, _ := engine.Render(p, data)
//	    // write content to disk
//	}
//
// Only files are returned (directories are excluded).
// Traversal is recursive — nested subdirectories are included.
func (e *Engine) ListTemplates(subDir string) ([]string, error) {
	root := e.baseDir
	if subDir != "" {
		root = e.baseDir + "/" + subDir
	}
	var paths []string

	err := fs.WalkDir(e.fs, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			// Return path relative to baseDir (strip baseDir prefix)
			rel := path[len(e.baseDir)+1:]
			paths = append(paths, rel)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listing templates in %s: %w", root, err)
	}

	return paths, nil
}
