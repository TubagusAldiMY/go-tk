package new

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/TubagusAldiMY/go-tk/internal/config"
	"github.com/TubagusAldiMY/go-tk/internal/generator"
	"github.com/TubagusAldiMY/go-tk/internal/ui"
)

// TemplatesFS must be set by the cmd package via go:embed.
// It is declared here as a variable so the command layer can inject it.
var TemplatesFS embed.FS

// NewProjectData is the data passed to every project template.
type NewProjectData struct {
	ProjectName string
	ModulePath  string
	GoVersion   string
	Framework   string
	Database    string
	ORM         string
	Auth        string
	HasDocker   bool
	HasCICD     bool
	Year        int
}

// GenerateProject creates a new project from templates into targetDir.
// If dryRun is true, it prints what would be created without writing files.
func GenerateProject(opts *ProjectOptions, targetDir string, dryRun bool) error {
	templateDir := templateDirForStack(opts.Framework, opts.Database)
	engine := generator.NewEngine(TemplatesFS, "project")

	data := NewProjectData{
		ProjectName: opts.ProjectName,
		ModulePath:  opts.ModulePath,
		GoVersion:   "1.22",
		Framework:   opts.Framework,
		Database:    opts.Database,
		ORM:         opts.ORM,
		Auth:        opts.Auth,
		HasDocker:   opts.HasDocker,
		HasCICD:     opts.HasCICD,
		Year:        time.Now().Year(),
	}

	templates, err := engine.ListTemplates(templateDir)
	if err != nil {
		return fmt.Errorf("listing templates: %w", err)
	}

	if dryRun {
		ui.PrintSection("Dry run — files that would be created")
	} else if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("creating project directory: %w", err)
	}

	for _, tmplPath := range templates {
		if err := renderTemplate(engine, opts, templateDir, targetDir, tmplPath, dryRun, data); err != nil {
			return err
		}
	}

	if dryRun {
		return nil
	}
	return finalizeProject(opts, targetDir)
}

// renderTemplate processes a single template file, applying CICD gating and dry-run logic.
func renderTemplate(engine *generator.Engine, opts *ProjectOptions, templateDir, targetDir, tmplPath string, dryRun bool, data NewProjectData) error {
	if !opts.HasCICD && strings.Contains(tmplPath, ".github/workflows/") {
		return nil
	}
	outPath := resolveOutputPath(targetDir, tmplPath, templateDir)
	if dryRun {
		ui.PrintDryRun(outPath)
		return nil
	}
	return processTemplateFile(engine, tmplPath, outPath, data)
}

// processTemplateFile renders a single template and writes it atomically to outPath.
func processTemplateFile(engine *generator.Engine, tmplPath, outPath string, data NewProjectData) error {
	if strings.HasSuffix(tmplPath, ".gitkeep.tmpl") {
		return generator.EnsureDir(filepath.Dir(outPath))
	}
	if generator.FileExists(outPath) {
		ui.PrintFileSkipped(outPath)
		return nil
	}

	rendered, err := engine.Render(tmplPath, data)
	if err != nil {
		return fmt.Errorf("rendering %s: %w", tmplPath, err)
	}

	if generator.IsGoFile(outPath) {
		if formatted, err := generator.FormatGo(rendered); err == nil {
			rendered = formatted
		}
	}

	if err := generator.WriteAtomic(outPath, rendered, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outPath, err)
	}
	ui.PrintFileCreated(outPath)

	if generator.IsGoFile(outPath) {
		_ = generator.RunGoimports(outPath)
	}
	return nil
}

// finalizeProject writes gotk.yaml and validates the generated project compiles.
func finalizeProject(opts *ProjectOptions, targetDir string) error {
	if err := writeGotkYAML(opts, targetDir); err != nil {
		return err
	}
	if err := validateProjectBuild(targetDir); err != nil {
		return fmt.Errorf("build validation failed — files written to %s but project does not compile: %w", targetDir, err)
	}
	return nil
}

// validateProjectBuild runs go mod tidy and go build ./... in targetDir.
// go mod tidy failure is reported as a warning but does not prevent go build from running,
// since tidy can fail in offline/network-restricted environments while the generated code
// itself may still be structurally valid. Returns an error only when go build fails.
func validateProjectBuild(targetDir string) error {
	ui.PrintSection("Validating generated project")

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = targetDir
	if out, err := tidy.CombinedOutput(); err != nil {
		if !ui.Quiet {
			fmt.Printf("  ! go mod tidy: %s\n", string(out))
			ui.PrintHint("Network may be unavailable — continuing to build check anyway.")
		}
	}

	build := exec.Command("go", "build", "./...")
	build.Dir = targetDir
	if out, err := build.CombinedOutput(); err != nil {
		return fmt.Errorf("go build failed:\n%s", string(out))
	}

	ui.PrintDone("Generated project compiles successfully")
	return nil
}

// templateDirForStack returns the template subdirectory for the given stack.
func templateDirForStack(framework, database string) string {
	return framework + "-" + database
}

// resolveOutputPath converts a template path like
// "gin-postgres/cmd/api/main.go.tmpl" → "<targetDir>/cmd/api/main.go"
func resolveOutputPath(targetDir, tmplPath, templateDir string) string {
	// Strip the "gin-postgres/" prefix.
	rel := strings.TrimPrefix(tmplPath, templateDir+"/")
	// Strip .tmpl extension.
	rel = strings.TrimSuffix(rel, ".tmpl")
	return filepath.Join(targetDir, rel)
}

// writeGotkYAML writes the gotk.yaml config file into the project root.
func writeGotkYAML(opts *ProjectOptions, targetDir string) error {
	cfg := config.DefaultConfig(opts.ProjectName, opts.ModulePath)
	cfg.Stack.Framework = opts.Framework
	cfg.Stack.Database = opts.Database
	cfg.Stack.ORM = opts.ORM
	cfg.Stack.Auth = opts.Auth

	content := fmt.Sprintf(`version: 1

project:
  name: %s
  module: %s

stack:
  framework: %s
  database: %s
  orm: %s
  auth: %s

paths:
  handlers: internal/interfaces/http/handler
  services: internal/application/usecase
  repos: internal/infrastructure/repository
  migrations: internal/infrastructure/database/migrations
  entities: internal/domain/entity

generate:
  soft_delete: true
  timestamps: true
  swagger: false

migrate:
  driver: %s
  dsn: ${DATABASE_URL}
`,
		cfg.Project.Name,
		cfg.Project.Module,
		cfg.Stack.Framework,
		cfg.Stack.Database,
		cfg.Stack.ORM,
		cfg.Stack.Auth,
		cfg.Stack.Database,
	)

	outPath := filepath.Join(targetDir, "gotk.yaml")
	if generator.FileExists(outPath) {
		ui.PrintFileSkipped(outPath)
		return nil
	}

	if err := generator.WriteAtomic(outPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing gotk.yaml: %w", err)
	}
	ui.PrintFileCreated(outPath)
	return nil
}

// ValidateFS ensures the embed.FS contains templates for the requested stack.
func ValidateFS(framework, database string) error {
	dir := "project/" + framework + "-" + database
	entries, err := fs.ReadDir(TemplatesFS, dir)
	if err != nil || len(entries) == 0 {
		return fmt.Errorf("no templates found for stack %s-%s", framework, database)
	}
	return nil
}
