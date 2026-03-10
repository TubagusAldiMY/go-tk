package new

import (
	"github.com/TubagusAldiMY/go-tk/internal/config"
	"github.com/TubagusAldiMY/go-tk/internal/ui"
)

// ProjectOptions holds the answers from interactive prompts.
type ProjectOptions struct {
	ProjectName string
	ModulePath  string
	Framework   string
	Database    string
	ORM         string
	Auth        string
	HasDocker   bool
	HasCICD     bool
}

// RunInteractivePrompts runs the Bubbletea TUI to collect project options.
// Returns the filled ProjectOptions or an error if the user cancelled.
func RunInteractivePrompts(projectName string) (*ProjectOptions, error) {
	opts := &ProjectOptions{
		ProjectName: projectName,
		ModulePath:  "github.com/username/" + projectName,
	}

	ui.PrintSection("Configure your project")

	// Framework
	framework, err := ui.RunSelector("Framework", []ui.SelectOption{
		{Label: "Gin", Value: config.FrameworkGin, Desc: "Fast HTTP framework, most popular"},
		{Label: "Fiber", Value: "fiber", Desc: "Express-inspired, high performance"},
	})
	if err != nil || framework == "" {
		return nil, ErrPromptCancelled
	}
	opts.Framework = framework

	// Database
	database, err := ui.RunSelector("Database", []ui.SelectOption{
		{Label: "PostgreSQL", Value: config.DatabasePostgres, Desc: "Recommended — full ACID compliance"},
		{Label: "MySQL", Value: config.DatabaseMySQL, Desc: "Widely adopted relational database"},
	})
	if err != nil || database == "" {
		return nil, ErrPromptCancelled
	}
	opts.Database = database

	// ORM
	orm, err := ui.RunSelector("ORM", []ui.SelectOption{
		{Label: "GORM", Value: config.ORMGorm, Desc: "Default — struct-based, easy to use"},
		{Label: "sqlc", Value: config.ORMSqlc, Desc: "Type-safe SQL — requires SQL knowledge"},
	})
	if err != nil || orm == "" {
		return nil, ErrPromptCancelled
	}
	opts.ORM = orm

	// Auth
	auth, err := ui.RunSelector("Authentication", []ui.SelectOption{
		{Label: "JWT", Value: config.AuthJWT, Desc: "Stateless token-based auth"},
		{Label: "None", Value: config.AuthNone, Desc: "Skip — add auth later manually"},
	})
	if err != nil || auth == "" {
		return nil, ErrPromptCancelled
	}
	opts.Auth = auth

	// Docker
	docker, err := ui.RunConfirm("Include Docker + docker-compose?")
	if err != nil {
		return nil, ErrPromptCancelled
	}
	opts.HasDocker = docker

	// CI/CD
	cicd, err := ui.RunConfirm("Include GitHub Actions CI/CD?")
	if err != nil {
		return nil, ErrPromptCancelled
	}
	opts.HasCICD = cicd

	return opts, nil
}
