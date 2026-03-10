// Command go-tk is the CLI entry point for the go-tk backend toolkit.
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
	newcmd.TemplatesFS = gotktmpl.FS
	crudcmd.TemplatesFS = gotktmpl.FS

	root := buildRootCmd()

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func buildRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "go-tk",
		Short: "Go Backend Toolkit — scaffold, generate, migrate, analyze",
		Long: `go-tk is a CLI toolkit for Go backend developers.

It eliminates the friction of starting and maintaining Go backend projects
by providing opinionated code generation, migration management, and analysis.

Learn more: https://github.com/TubagusAldiMY/go-tk`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.Version = fmt.Sprintf("%s (commit: %s, built: %s)", build.Version, build.Commit, build.Date)
	root.SetVersionTemplate("go-tk version {{.Version}}\n")

	root.AddCommand(newcmd.NewCmd())
	root.AddCommand(gencmd.GenCmd())
	root.AddCommand(migratecmd.MigrateCmd())
	root.AddCommand(envcmd.EnvCmd())
	root.AddCommand(testcmd.TestCmd())
	root.AddCommand(analyzecmd.AnalyzeCmd())

	return root
}
