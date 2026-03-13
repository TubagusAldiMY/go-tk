// Package crud implements the "go-tk gen crud" command.
package crud

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/TubagusAldiMY/go-tk/internal/config"
	"github.com/TubagusAldiMY/go-tk/internal/ui"
)

// CrudCmd returns the cobra.Command for "go-tk gen crud".
func CrudCmd() *cobra.Command {
	var (
		flagFields     string
		flagSoftDelete bool
		flagTimestamps bool
		flagDryRun     bool
		flagForce      bool
		flagSkip       bool
	)

	cmd := &cobra.Command{
		Use:   "crud <EntityName>",
		Short: "Generate CRUD files for an entity (handler, usecase, repository, dto, migration)",
		Long: `Generate all CRUD layers for a domain entity following Clean Architecture.

Each run creates up to 7 files:
  • internal/domain/entity/<entity>.go
  • internal/domain/repository/<entity>_repository.go  (interface)
  • internal/infrastructure/repository/<entity>_repo.go (GORM impl)
  • internal/application/usecase/<entity>_usecase.go
  • internal/interfaces/http/handler/<entity>_handler.go
  • internal/interfaces/dto/<entity>_dto.go
  • internal/infrastructure/database/migrations/<ts>_create_<table>.sql

Flags:
  --skip    Only generate files that don't exist (skip existing)
  --force   Overwrite all existing files

Examples:
  go-tk gen crud Product
  go-tk gen crud Product --fields='name:string,price:float64,stock:int,active:bool'
  go-tk gen crud Order --fields='total:float64,status:string?' --dry-run
  go-tk gen crud User --skip    # Only create missing files`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate mutually exclusive flags
			if flagForce && flagSkip {
				return fmt.Errorf("--force and --skip are mutually exclusive")
			}
			return runCrud(args[0], &crudFlags{
				fields:     flagFields,
				softDelete: flagSoftDelete,
				timestamps: flagTimestamps,
				dryRun:     flagDryRun,
				force:      flagForce,
				skip:       flagSkip,
			})
		},
	}

	cmd.Flags().StringVar(&flagFields, "fields", "", `Comma-separated field definitions: "name:type[?],...". Supported types: string, int, int64, float64, bool`)
	cmd.Flags().BoolVar(&flagSoftDelete, "soft-delete", true, "Add deleted_at column (soft delete)")
	cmd.Flags().BoolVar(&flagTimestamps, "timestamps", true, "Add created_at/updated_at columns")
	cmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Preview files that would be created without writing")
	cmd.Flags().BoolVar(&flagForce, "force", false, "Overwrite existing files")
	cmd.Flags().BoolVar(&flagSkip, "skip", false, "Only generate files that don't exist (skip existing)")

	return cmd
}

type crudFlags struct {
	fields     string
	softDelete bool
	timestamps bool
	dryRun     bool
	force      bool
	skip       bool
}

func runCrud(entityName string, flags *crudFlags) error {
	fmt.Println()
	fmt.Println(ui.Banner())
	fmt.Println()

	// Load project config from gotk.yaml.
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	cfg, err := config.Load(cwd)
	if err != nil {
		return ErrNoConfig
	}

	// Override config with flags.
	cfg.Generate.SoftDelete = flags.softDelete
	cfg.Generate.Timestamps = flags.timestamps

	// Parse field definitions from --fields flag.
	fields, err := ParseFields(flags.fields)
	if err != nil {
		return fmt.Errorf("parsing --fields: %w", err)
	}

	if len(fields) == 0 && !flags.dryRun {
		ui.PrintHint("No --fields provided — generating entity with ID, timestamps only.")
		ui.PrintHint("You can add fields manually or re-run with --fields='name:type,...'")
		fmt.Println()
	}

	return Generate(entityName, cfg, fields, flags.force, flags.skip, flags.dryRun)
}
