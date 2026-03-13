// Command go-tk is the CLI entry point for the go-tk backend toolkit.
//
// go-tk is a production-ready CLI generator for Go backend projects, analogous to
// Laravel Artisan or Rails CLI. It scaffolds projects with Clean Architecture,
// OWASP compliance, and opinionated security defaults.
//
// Core capabilities:
//   - Project scaffolding (go-tk new) with 4 stack combinations (Gin/Fiber × PostgreSQL/MySQL)
//   - CRUD generation (go-tk gen crud) with full Clean Architecture layers
//   - Database migration management (go-tk migrate) via golang-migrate wrapper
//   - Environment validation (go-tk env) for .env consistency
//   - HTTP route testing (go-tk test) with auto-discovery
//   - Static code analysis (go-tk analyze) for security and quality issues
//
// Architecture:
// The CLI uses Cobra for command routing, Bubbletea for interactive TUI,
// and embeds all templates via go:embed for zero external dependencies.
// Version info (Version, Commit, Date) is injected at build time via ldflags.
//
// See CLAUDE.md and AGENTS.md for architectural decisions and compliance standards.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/TubagusAldiMY/go-tk/cmd/go-tk/build"
	analyzecmd "github.com/TubagusAldiMY/go-tk/internal/command/analyze"
	envcmd "github.com/TubagusAldiMY/go-tk/internal/command/env"
	gencmd "github.com/TubagusAldiMY/go-tk/internal/command/gen"
	crudcmd "github.com/TubagusAldiMY/go-tk/internal/command/gen/crud"
	migratecmd "github.com/TubagusAldiMY/go-tk/internal/command/migrate"
	newcmd "github.com/TubagusAldiMY/go-tk/internal/command/new"
	testcmd "github.com/TubagusAldiMY/go-tk/internal/command/test"
	gotktmpl "github.com/TubagusAldiMY/go-tk/templates"
)

func main() {
	// Inject the embedded FS into commands that generate files.
	// This allows new and crud commands to access template files
	// without requiring external template directories at runtime.
	// Templates are compiled into the binary via go:embed in templates/embed.go.
	newcmd.TemplatesFS = gotktmpl.FS
	crudcmd.TemplatesFS = gotktmpl.FS

	root := buildRootCmd()

	// Execute the CLI — errors are printed to stderr.
	// SilenceUsage/SilenceErrors in root command prevent Cobra
	// from printing redundant error output.
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// buildRootCmd constructs the root Cobra command with all subcommands attached.
//
// Command hierarchy:
//   go-tk new [name]                  — Scaffold new project (interactive or flags)
//   go-tk gen crud <Entity>           — Generate full CRUD (entity, repo, usecase, handler, DTO, migrations)
//   go-tk migrate [up|down|status|...]— Database migration management
//   go-tk env [validate|sync|...]     — Environment variable validation
//   go-tk test                        — Auto-generate and run HTTP tests
//   go-tk analyze                     — Static analysis (7 checks: errors, N+1, auth, etc.)
//
// Version information (from ldflags):
//   Build: make build injects Version, Commit, Date via ldflags
//   Example: go build -ldflags "-X .../build.Version=v1.0.0"
//
// Silence flags prevent duplicate error output — we handle errors explicitly in main().
func buildRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "go-tk",
		Short: "Go Backend Toolkit — scaffold, generate, migrate, analyze",
		Long: `go-tk is a CLI toolkit for Go backend developers.

It eliminates the friction of starting and maintaining Go backend projects
by providing opinionated code generation, migration management, and analysis.

Learn more: https://github.com/TubagusAldiMY/go-tk`,
		SilenceUsage:  true,  // Don't print usage on error (only on --help)
		SilenceErrors: true,  // We print errors manually in main()
	}

	// Version template uses values injected at build time via ldflags.
	// See Makefile LDFLAGS for injection points.
	root.Version = fmt.Sprintf("%s (commit: %s, built: %s)", build.Version, build.Commit, build.Date)
	root.SetVersionTemplate("go-tk version {{.Version}}\n")

	// Register all subcommands — order matches help output.
	root.AddCommand(newcmd.NewCmd())
	root.AddCommand(gencmd.GenCmd())
	root.AddCommand(migratecmd.MigrateCmd())
	root.AddCommand(envcmd.EnvCmd())
	root.AddCommand(testcmd.TestCmd())
	root.AddCommand(analyzecmd.AnalyzeCmd())

	return root
}
