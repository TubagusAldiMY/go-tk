// Package gen is the parent command for all "go-tk gen *" sub-commands.
package gen

import (
	"github.com/spf13/cobra"

	"github.com/TubagusAldiMY/go-tk/internal/command/gen/crud"
)

// GenCmd returns the cobra.Command for "go-tk gen".
func GenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen",
		Short: "Code generators (crud, ...)",
		Long:  `Generate production-ready code for your Go backend project.`,
	}

	cmd.AddCommand(crud.CrudCmd())

	return cmd
}
