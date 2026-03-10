package config

// Config represents the full gotk.yaml structure.
type Config struct {
	Version  int            `mapstructure:"version"  yaml:"version"`
	Project  ProjectConfig  `mapstructure:"project"  yaml:"project"`
	Stack    StackConfig    `mapstructure:"stack"    yaml:"stack"`
	Paths    PathsConfig    `mapstructure:"paths"    yaml:"paths"`
	Generate GenerateConfig `mapstructure:"generate" yaml:"generate"`
	Migrate  MigrateConfig  `mapstructure:"migrate"  yaml:"migrate"`
}

// ProjectConfig holds project identity fields.
type ProjectConfig struct {
	Name   string `mapstructure:"name"   yaml:"name"`
	Module string `mapstructure:"module" yaml:"module"`
}

// StackConfig describes the chosen technology stack.
type StackConfig struct {
	Framework string `mapstructure:"framework" yaml:"framework"` // gin | fiber
	Database  string `mapstructure:"database"  yaml:"database"`  // postgres | mysql
	ORM       string `mapstructure:"orm"       yaml:"orm"`       // gorm | sqlc
	Auth      string `mapstructure:"auth"      yaml:"auth"`      // jwt | none
}

// PathsConfig defines where generated files land in the target project.
type PathsConfig struct {
	Handlers   string `mapstructure:"handlers"   yaml:"handlers"`
	Services   string `mapstructure:"services"   yaml:"services"`
	Repos      string `mapstructure:"repos"      yaml:"repos"`
	Migrations string `mapstructure:"migrations" yaml:"migrations"`
	Entities   string `mapstructure:"entities"   yaml:"entities"`
}

// GenerateConfig holds code generation preferences.
type GenerateConfig struct {
	SoftDelete bool `mapstructure:"soft_delete" yaml:"soft_delete"`
	Timestamps bool `mapstructure:"timestamps"  yaml:"timestamps"`
	Swagger    bool `mapstructure:"swagger"     yaml:"swagger"`
}

// MigrateConfig holds migration runner settings.
type MigrateConfig struct {
	Driver string `mapstructure:"driver" yaml:"driver"`
	DSN    string `mapstructure:"dsn"    yaml:"dsn"`
}

// Stack / auth option constants.
const (
	FrameworkGin   = "gin"
	FrameworkFiber = "fiber"

	DatabasePostgres = "postgres"
	DatabaseMySQL    = "mysql"

	ORMGorm = "gorm"
	ORMSqlc = "sqlc"

	AuthJWT  = "jwt"
	AuthNone = "none"
)

// DefaultConfig returns an opinionated default configuration for new projects.
func DefaultConfig(name, module string) Config {
	return Config{
		Version: 1,
		Project: ProjectConfig{Name: name, Module: module},
		Stack: StackConfig{
			Framework: FrameworkGin,
			Database:  DatabasePostgres,
			ORM:       ORMGorm,
			Auth:      AuthJWT,
		},
		Paths:    DefaultPaths(),
		Generate: GenerateConfig{SoftDelete: true, Timestamps: true, Swagger: false},
		Migrate: MigrateConfig{
			Driver: DatabasePostgres,
			DSN:    "${DATABASE_URL}",
		},
	}
}

// DefaultPaths returns the canonical path layout.
func DefaultPaths() PathsConfig {
	return PathsConfig{
		Handlers:   "internal/interfaces/http/handler",
		Services:   "internal/application/usecase",
		Repos:      "internal/infrastructure/repository",
		Migrations: "internal/infrastructure/database/migrations",
		Entities:   "internal/domain/entity",
	}
}
