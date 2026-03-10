// Package migrate implements the "go-tk migrate" command family.
package migrate

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	gmigrate "github.com/golang-migrate/migrate/v4"

	"github.com/TubagusAldiMY/go-tk/internal/config"
	"github.com/TubagusAldiMY/go-tk/internal/ui"
)

// MigrateCmd returns the cobra.Command for "go-tk migrate".
func MigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Database migration management (up, down, status, create, validate)",
		Long: `Manage database migrations for your go-tk project.

Wraps golang-migrate with a clean status dashboard, dry-run preview,
and automatic .env loading.

Reads database config from gotk.yaml and DATABASE_URL from .env.`,
	}

	cmd.AddCommand(upCmd())
	cmd.AddCommand(downCmd())
	cmd.AddCommand(statusCmd())
	cmd.AddCommand(createCmd())
	cmd.AddCommand(validateCmd())

	return cmd
}

func upCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "up",
		Short: "Apply all pending migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withRunner(func(r *Runner, cfg *config.Config) error {
				if dryRun {
					return dryRunUp(r)
				}
				ui.PrintSection("Applying migrations")
				if err := r.Up(); err != nil {
					return err
				}
				ui.PrintDone("All migrations applied.")
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print SQL without executing")
	return cmd
}

func downCmd() *cobra.Command {
	var steps int

	cmd := &cobra.Command{
		Use:   "down",
		Short: "Roll back migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withRunner(func(r *Runner, cfg *config.Config) error {
				ui.PrintSection(fmt.Sprintf("Rolling back %d migration(s)", steps))
				if err := r.Down(steps); err != nil {
					return err
				}
				ui.PrintDone("Rollback complete.")
				return nil
			})
		},
	}
	cmd.Flags().IntVar(&steps, "steps", 1, "Number of migrations to roll back")
	return cmd
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show applied and pending migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withRunner(func(r *Runner, cfg *config.Config) error {
				version, dirty, err := r.Version()
				if err != nil && !errors.Is(err, gmigrate.ErrNilVersion) {
					return fmt.Errorf("reading migration version: %w", err)
				}

				statuses, err := GetStatus(r.migrationsDir, version, dirty)
				if err != nil {
					return err
				}

				PrintStatusTable(statuses, cfg.Stack.Database, cfg.Migrate.DSN, version, dirty)
				return nil
			})
		},
	}
}

func createCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new migration file pair (.up.sql + .down.sql)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, _ := os.Getwd()
			cfg, err := config.Load(cwd)
			if err != nil {
				return config.ErrConfigNotFound
			}

			r := &Runner{cfg: cfg, migrationsDir: cfg.Paths.Migrations}
			ui.PrintSection("Creating migration: " + args[0])
			if err := r.Create(args[0]); err != nil {
				return err
			}
			ui.PrintDone("Migration files created.")
			ui.PrintHint("Edit the .up.sql and .down.sql files, then run: go-tk migrate up")
			return nil
		},
	}
}

func validateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate migration file pairs",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, _ := os.Getwd()
			cfg, err := config.Load(cwd)
			if err != nil {
				return config.ErrConfigNotFound
			}

			r := &Runner{cfg: cfg, migrationsDir: cfg.Paths.Migrations}
			ui.PrintSection("Validating migrations")
			return r.Validate()
		},
	}
}

// withRunner loads project config, creates a Runner, and executes fn.
func withRunner(fn func(r *Runner, cfg *config.Config) error) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	cfg, err := config.Load(cwd)
	if err != nil {
		return config.ErrConfigNotFound
	}
	r, err := NewRunner(cfg, cwd)
	if err != nil {
		return err
	}
	return fn(r, cfg)
}

// dryRunUp prints the SQL from pending up-migration files without executing them.
func dryRunUp(r *Runner) error {
	version, _, _ := r.Version()

	statuses, err := GetStatus(r.migrationsDir, version, false)
	if err != nil {
		return err
	}

	ui.PrintSection("Dry run — SQL that would be applied")
	hasAny := false
	for _, s := range statuses {
		if s.Applied {
			continue
		}
		hasAny = true
		content, err := os.ReadFile(r.migrationsDir + "/" + s.Filename)
		if err != nil {
			ui.PrintError("Cannot read " + s.Filename + ": " + err.Error())
			continue
		}
		fmt.Printf("\n-- %s\n%s\n", s.Filename, string(content))
	}
	if !hasAny {
		ui.PrintHint("No pending migrations.")
	}
	return nil
}
