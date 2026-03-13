// Package crud implements the "go-tk gen crud" command.
//
// This package generates a complete CRUD implementation following Clean Architecture:
//
//	Layer 1 (Domain):         entity, repository interface
//	Layer 2 (Application):    use case (business logic)
//	Layer 3 (Infrastructure): repository implementation (GORM)
//	Layer 4 (Delivery):       HTTP handler (Gin or Fiber), DTO
//	Database:                 migrations (up/down) for PostgreSQL or MySQL
//
// Total files generated: 8 per entity
//  1. internal/domain/entity/{entity}.go
//  2. internal/domain/repository/{entity}_repository.go (interface)
//  3. internal/infrastructure/repository/{entity}_repo.go (GORM impl)
//  4. internal/application/usecase/{entity}_usecase.go
//  5. internal/interfaces/http/handler/{entity}_handler.go (Gin or Fiber)
//  6. internal/interfaces/dto/{entity}_dto.go
//  7. migrations/{timestamp}_create_{entities}.up.sql (Postgres or MySQL)
//  8. migrations/{timestamp}_create_{entities}.down.sql
//
// Architecture Decisions:
//   - Dual handler templates (Gin vs Fiber) — selected via gotk.yaml stack.framework
//   - Dual migration templates (Postgres vs MySQL) — selected via gotk.yaml stack.database
//   - Timestamps are idempotent within same second (clock-based, not counter-based)
//   - Soft delete is opt-in via gotk.yaml generate.soft_delete
//
// Idempotency:
//   - Files are skipped if they exist (unless --force)
//   - Running same command twice is safe (no duplicates, no corruption)
//   - Migration timestamps are deterministic (same second → same filename)
//
// Thread safety: Not concurrent-safe (creates files in sequence).
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
// This is set in cmd/go-tk/main.go via crudcmd.TemplatesFS = gotktmpl.FS.
// Do NOT initialize here — embedding must happen at main package level.
var TemplatesFS embed.FS

// CRUDData is the template data for all CRUD file templates.
//
// Naming conventions:
//
//	EntityName   — PascalCase (User, ProductCategory)
//	EntityNameLC — camelCase (user, productCategory) — for variable names
//	EntityNamePL — lowercase plural (users, productcategories) — for package names
//	TableName    — snake_case plural (users, product_categories) — for SQL
//
// This struct is passed to all 8 templates and must contain ALL data needed.
// Do NOT add conditional logic in templates — add fields here and branch in Go.
type CRUDData struct {
	ModulePath   string     // Go module path (e.g. github.com/user/myapp)
	EntityName   string     // PascalCase: "Product"
	EntityNameLC string     // camelCase:  "product"
	EntityNamePL string     // plural:     "products"
	TableName    string     // snake_case plural: "products"
	Fields       []FieldDef // Entity fields with type, DB type, validation
	SoftDelete   bool       // Whether to include deleted_at column
	Timestamps   bool       // Whether to include created_at/updated_at
	MigrationVer string     // Timestamp: "20260309120000"
	GeneratedAt  string     // Human date: "2026-03-09"
	Framework    string     // "gin" | "fiber" (selects handler template)
	Database     string     // "postgres" | "mysql" (selects migration template + field types)
}

// GeneratedFile tracks a file that was (or would be) generated.
// Used for --dry-run preview and logging.
type GeneratedFile struct {
	TemplateName string // Template path (e.g. "crud/entity.go.tmpl")
	OutputPath   string // Target file path (e.g. "internal/domain/entity/product.go")
}

// Generate produces all CRUD files for the given entity.
//
// Workflow:
//  1. Build CRUDData from inputs (entity name, config, fields)
//  2. Select templates based on stack (Gin/Fiber × Postgres/MySQL)
//  3. For each template:
//     a. Check if output file exists (skip if not --force)
//     b. Render template with CRUDData
//     c. Format Go files (gofmt + goimports)
//     d. Write atomically to disk
//  4. Print next steps (route registration, dependency wiring)
//
// Parameters:
//
//	entityName — Entity name in any case (will be normalized to PascalCase)
//	cfg        — Project config from gotk.yaml (paths, stack, options)
//	fields     — Field definitions (from CLI flags or interactive prompt)
//	force      — Overwrite existing files (default: skip existing)
//	dryRun     — Print what would be generated without writing files
//
// Error handling:
//   - Template render error → fail immediately (don't write partial files)
//   - File write error → fail immediately (atomic write prevents corruption)
//   - Formatting error → log warning, continue (generated code may need manual fmt)
//
// Idempotency:
//
//	Running this function twice with same inputs:
//	  force=false  → second run skips all files (no changes)
//	  force=true   → second run overwrites all files (identical output)
//	  dryRun=true  → never writes files (safe to run multiple times)
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
