package generator

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"text/template"
)

// Engine renders text/template templates from an embedded filesystem.
type Engine struct {
	fs      embed.FS
	baseDir string
	funcs   template.FuncMap
}

// NewEngine creates a new template Engine backed by the given embed.FS.
// baseDir is the root directory within the FS that contains templates
// (e.g. "templates").
func NewEngine(fsys embed.FS, baseDir string) *Engine {
	return &Engine{
		fs:      fsys,
		baseDir: baseDir,
		funcs:   defaultFuncMap(),
	}
}

// Render parses the template at templatePath (relative to baseDir) and
// executes it with data, returning the rendered bytes.
func (e *Engine) Render(templatePath string, data any) ([]byte, error) {
	fullPath := e.baseDir + "/" + templatePath

	src, err := fs.ReadFile(e.fs, fullPath)
	if err != nil {
		return nil, fmt.Errorf("reading template %s: %w", fullPath, err)
	}

	tmpl, err := template.New(templatePath).Funcs(e.funcs).Parse(string(src))
	if err != nil {
		return nil, fmt.Errorf("parsing template %s: %w", fullPath, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template %s: %w", fullPath, err)
	}

	return buf.Bytes(), nil
}

// RenderString renders a raw template string (not from the FS) with data.
// Useful for inline templates (e.g. gotk.yaml content).
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
			// Return path relative to baseDir
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
