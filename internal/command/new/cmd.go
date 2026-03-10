// Package new implements the "go-tk new" command.
package new

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TubagusAldiMY/go-tk/internal/config"
	"github.com/TubagusAldiMY/go-tk/internal/ui"
)

// NewCmd returns the cobra.Command for "go-tk new".
func NewCmd() *cobra.Command {
	var (
		flagFramework string
		flagDB        string
		flagORM       string
		flagAuth      string
		flagDocker    bool
		flagCICD      bool
		flagDryRun    bool
		flagModule    string
	)

	cmd := &cobra.Command{
		Use:   "new <project-name>",
		Short: "Scaffold a new Go backend project with Clean Architecture",
		Long: `Scaffold a new Go backend project following Clean Architecture,
OWASP security standards, and 12-Factor App principles.

Examples:
  # Interactive mode (recommended)
  go-tk new my-api

  # Non-interactive (CI/CD friendly)
  go-tk new my-api --framework=gin --db=postgres --auth=jwt --docker`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNew(args[0], &newFlags{
				framework: flagFramework,
				db:        flagDB,
				orm:       flagORM,
				auth:      flagAuth,
				docker:    flagDocker,
				cicd:      flagCICD,
				dryRun:    flagDryRun,
				module:    flagModule,
			})
		},
	}

	cmd.Flags().StringVar(&flagFramework, "framework", "", "HTTP framework: gin|fiber")
	cmd.Flags().StringVar(&flagDB, "db", "", "Database: postgres|mysql")
	cmd.Flags().StringVar(&flagORM, "orm", config.ORMGorm, "ORM: gorm|sqlc")
	cmd.Flags().StringVar(&flagAuth, "auth", config.AuthJWT, "Auth: jwt|none")
	cmd.Flags().BoolVar(&flagDocker, "docker", true, "Include Dockerfile and docker-compose.yml")
	cmd.Flags().BoolVar(&flagCICD, "cicd", false, "Include GitHub Actions workflow")
	cmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Preview files that would be created without writing")
	cmd.Flags().StringVar(&flagModule, "module", "", "Go module path (default: github.com/<user>/<name>)")

	return cmd
}

type newFlags struct {
	framework, db, orm, auth, module string
	docker, cicd, dryRun             bool
}

func runNew(projectName string, flags *newFlags) error {
	if err := validateProjectName(projectName); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println(ui.Banner())
	fmt.Println()

	var opts *ProjectOptions
	var err error

	// Non-interactive if both framework and db are supplied via flags.
	if flags.framework != "" && flags.db != "" {
		opts = &ProjectOptions{
			ProjectName: projectName,
			ModulePath:  resolveModulePath(flags.module, projectName),
			Framework:   flags.framework,
			Database:    flags.db,
			ORM:         flags.orm,
			Auth:        flags.auth,
			HasDocker:   flags.docker,
			HasCICD:     flags.cicd,
		}
	} else {
		opts, err = RunInteractivePrompts(projectName)
		if err != nil {
			return fmt.Errorf("prompts: %w", err)
		}
		if flags.module != "" {
			opts.ModulePath = flags.module
		}
	}

	targetDir := filepath.Join(".", projectName)

	if !flags.dryRun {
		// Check if directory already exists and is non-empty.
		if entries, err := os.ReadDir(targetDir); err == nil && len(entries) > 0 {
			return fmt.Errorf("%w: %s — use a different name or remove the directory", ErrProjectExists, targetDir)
		}
	}

	// Validate the requested stack has templates.
	if err := ValidateFS(opts.Framework, opts.Database); err != nil {
		return err
	}

	ui.PrintSection("Generating project: " + projectName)

	if err := GenerateProject(opts, targetDir, flags.dryRun); err != nil {
		ui.PrintError("Generation failed: " + err.Error())
		return err
	}

	if flags.dryRun {
		fmt.Println()
		ui.PrintHint("Run without --dry-run to create the project.")
		return nil
	}

	printNextSteps(projectName, opts)
	return nil
}

func printNextSteps(name string, opts *ProjectOptions) {
	fmt.Println()
	ui.PrintDone(fmt.Sprintf("Project '%s' created successfully!", name))
	fmt.Println()
	ui.PrintSection("Next steps")
	ui.PrintHint(fmt.Sprintf("cd %s", name))
	ui.PrintHint("cp .env.example .env  # edit DATABASE_URL and JWT_SECRET")
	if opts.HasDocker {
		ui.PrintHint("docker compose up -d  # start PostgreSQL")
	}
	ui.PrintHint("go-tk migrate up      # run migrations")
	ui.PrintHint("go run ./cmd/api/     # start the server")
	fmt.Println()
}

func validateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	if strings.ContainsAny(name, " /\\:*?\"<>|") {
		return fmt.Errorf("project name contains invalid characters")
	}
	return nil
}

func resolveModulePath(flagModule, projectName string) string {
	if flagModule != "" {
		return flagModule
	}
	return "github.com/username/" + projectName
}
