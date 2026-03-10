package crud

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/TubagusAldiMY/go-tk/internal/config"
	"github.com/TubagusAldiMY/go-tk/internal/generator"
	"github.com/TubagusAldiMY/go-tk/internal/ui"
)

// TemplatesFS must be injected by the cmd layer (go:embed).
var TemplatesFS embed.FS

// CRUDData is the template data for all CRUD file templates.
type CRUDData struct {
	ModulePath   string
	EntityName   string // PascalCase: "Product"
	EntityNameLC string // lowercase:  "product"
	EntityNamePL string // plural:     "products"
	TableName    string // snake plural: "products"
	Fields       []FieldDef
	SoftDelete   bool
	Timestamps   bool
	MigrationVer string // timestamp: "20260309120000"
	GeneratedAt  string // human date: "2026-03-09"
	Framework    string // "gin" | "fiber"
	Database     string // "postgres" | "mysql"
}

// GeneratedFile tracks a file that was (or would be) generated.
type GeneratedFile struct {
	TemplateName string
	OutputPath   string
}

// Generate produces all CRUD files for the given entity.
func Generate(entityName string, cfg *config.Config, fields []FieldDef, force, dryRun bool) error {
	now := time.Now()
	data := CRUDData{
		ModulePath:   cfg.Project.Module,
		EntityName:   toPascalCase(entityName),
		EntityNameLC: strings.ToLower(entityName[:1]) + entityName[1:],
		EntityNamePL: toPlural(strings.ToLower(entityName)),
		TableName:    toPlural(toSnakeCase(entityName)),
		Fields:       fields,
		SoftDelete:   cfg.Generate.SoftDelete,
		Timestamps:   cfg.Generate.Timestamps,
		MigrationVer: now.Format("20060102150405"),
		GeneratedAt:  now.Format("2006-01-02"),
		Framework:    cfg.Stack.Framework,
		Database:     cfg.Stack.Database,
	}

	engine := generator.NewEngine(TemplatesFS, "crud")

	files := []GeneratedFile{
		{TemplateName: "entity.go.tmpl", OutputPath: filepath.Join(cfg.Paths.Entities, toSnakeCase(entityName)+".go")},
		{TemplateName: "repository_interface.go.tmpl", OutputPath: filepath.Join("internal/domain/repository", toSnakeCase(entityName)+"_repository.go")},
		{TemplateName: "repository_impl.go.tmpl", OutputPath: filepath.Join(cfg.Paths.Repos, toSnakeCase(entityName)+"_repo.go")},
		{TemplateName: "usecase.go.tmpl", OutputPath: filepath.Join(cfg.Paths.Services, toSnakeCase(entityName)+"_usecase.go")},
		{TemplateName: handlerTemplate(cfg.Stack.Framework), OutputPath: filepath.Join(cfg.Paths.Handlers, toSnakeCase(entityName)+"_handler.go")},
		{TemplateName: "dto.go.tmpl", OutputPath: filepath.Join("internal/interfaces/dto", toSnakeCase(entityName)+"_dto.go")},
		{TemplateName: migrationUpTemplate(cfg.Stack.Database), OutputPath: filepath.Join(cfg.Paths.Migrations, data.MigrationVer+"_create_"+data.TableName+".up.sql")},
		{TemplateName: "migration.down.sql.tmpl", OutputPath: filepath.Join(cfg.Paths.Migrations, data.MigrationVer+"_create_"+data.TableName+".down.sql")},
	}

	if dryRun {
		ui.PrintSection("Dry run — CRUD files for " + data.EntityName)
		for _, f := range files {
			ui.PrintDryRun(f.OutputPath)
		}
		return nil
	}

	ui.PrintSection("Generating CRUD: " + data.EntityName)

	for _, f := range files {
		if generator.FileExists(f.OutputPath) && !force {
			ui.PrintFileSkipped(f.OutputPath)
			continue
		}

		rendered, err := engine.Render(f.TemplateName, data)
		if err != nil {
			return fmt.Errorf("rendering %s: %w", f.TemplateName, err)
		}

		if generator.IsGoFile(f.OutputPath) {
			if formatted, err := generator.FormatGo(rendered); err == nil {
				rendered = formatted
			}
		}

		if err := generator.WriteAtomic(f.OutputPath, rendered, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", f.OutputPath, err)
		}
		ui.PrintFileCreated(f.OutputPath)

		if generator.IsGoFile(f.OutputPath) {
			_ = generator.RunGoimports(f.OutputPath)
		}
	}

	printCRUDNextSteps(data.EntityName, data.EntityNameLC, data.EntityNamePL)
	return nil
}

func printCRUDNextSteps(entity, entityLC, entityPL string) {
	fmt.Println()
	ui.PrintDone(fmt.Sprintf("CRUD for '%s' generated!", entity))
	fmt.Println()
	ui.PrintSection("Next steps")
	ui.PrintHint("Register routes in internal/interfaces/http/router.go:")
	fmt.Printf("  %s.RegisterRoutes(v1, %sHandler)\n", entityLC, entityLC)
	ui.PrintHint("Wire dependencies in cmd/api/main.go:")
	fmt.Printf("  %sRepo := repository.New%sRepository(db, log)\n", entityLC, entity)
	fmt.Printf("  %sUC   := usecase.New%sUseCase(%sRepo, log)\n", entityLC, entity, entityLC)
	fmt.Printf("  %sH    := handler.New%sHandler(%sUC, log)\n", entityLC, entity, entityLC)
	ui.PrintHint("Run migrations:")
	fmt.Println("  go-tk migrate up")
	fmt.Println()
}

// handlerTemplate returns the correct handler template name for the given framework.
func handlerTemplate(framework string) string {
	if framework == "fiber" {
		return "handler_fiber.go.tmpl"
	}
	return "handler_gin.go.tmpl"
}

// migrationUpTemplate returns the correct migration up template for the given database.
func migrationUpTemplate(database string) string {
	if database == "mysql" {
		return "migration.up.mysql.sql.tmpl"
	}
	return "migration.up.postgres.sql.tmpl"
}

// toPlural appends 's' or applies simple English plural rules.
func toPlural(s string) string {
	switch {
	case strings.HasSuffix(s, "s"), strings.HasSuffix(s, "x"),
		strings.HasSuffix(s, "z"), strings.HasSuffix(s, "ch"),
		strings.HasSuffix(s, "sh"):
		return s + "es"
	case strings.HasSuffix(s, "y") && len(s) > 1 && !isVowel(s[len(s)-2]):
		return s[:len(s)-1] + "ies"
	default:
		return s + "s"
	}
}

func isVowel(b byte) bool {
	return strings.ContainsRune("aeiou", rune(b))
}
