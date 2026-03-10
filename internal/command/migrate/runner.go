package migrate

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	gmigrate "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	gmysql "github.com/golang-migrate/migrate/v4/database/mysql"
	gpostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/TubagusAldiMY/go-tk/internal/config"
	"github.com/TubagusAldiMY/go-tk/internal/ui"
)

// Runner executes database migrations for a go-tk project.
type Runner struct {
	cfg           *config.Config
	migrationsDir string
	dsn           string
}

// NewRunner creates a Runner from project config.
// It loads .env from projectRoot to resolve DSN env var references.
func NewRunner(cfg *config.Config, projectRoot string) (*Runner, error) {
	// Load .env silently — env vars may already be set via CI/shell.
	_ = godotenv.Load(filepath.Join(projectRoot, ".env"))

	dsn := os.ExpandEnv(cfg.Migrate.DSN)
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		return nil, fmt.Errorf("database DSN not configured — set DATABASE_URL in .env or migrate.dsn in gotk.yaml")
	}

	return &Runner{
		cfg:           cfg,
		migrationsDir: filepath.Join(projectRoot, cfg.Paths.Migrations),
		dsn:           dsn,
	}, nil
}

// Up applies all pending migrations.
func (r *Runner) Up() error {
	m, cleanup, err := r.newMigrate()
	if err != nil {
		return err
	}
	defer cleanup()

	if err := m.Up(); err != nil && !errors.Is(err, gmigrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	if errors.Is(err, gmigrate.ErrNoChange) {
		ui.PrintHint("No pending migrations — database is already up to date.")
	}
	return nil
}

// Down rolls back the given number of migrations (minimum 1).
func (r *Runner) Down(steps int) error {
	if steps <= 0 {
		steps = 1
	}
	m, cleanup, err := r.newMigrate()
	if err != nil {
		return err
	}
	defer cleanup()

	if err := m.Steps(-steps); err != nil && !errors.Is(err, gmigrate.ErrNoChange) {
		return fmt.Errorf("migrate down: %w", err)
	}
	return nil
}

// Version returns the current migration version and dirty flag.
// Returns (0, false, nil) when no migrations have been applied yet.
func (r *Runner) Version() (uint, bool, error) {
	m, cleanup, err := r.newMigrate()
	if err != nil {
		return 0, false, err
	}
	defer cleanup()

	v, dirty, err := m.Version()
	if errors.Is(err, gmigrate.ErrNilVersion) {
		return 0, false, nil
	}
	return v, dirty, err
}

// Create creates a new migration file pair (<ts>_<name>.up.sql and .down.sql).
func (r *Runner) Create(name string) error {
	if err := os.MkdirAll(r.migrationsDir, 0o755); err != nil {
		return fmt.Errorf("creating migrations directory: %w", err)
	}

	ts := time.Now().Format("20060102150405")
	safeName := sanitizeMigrationName(name)
	base := filepath.Join(r.migrationsDir, ts+"_"+safeName)

	upContent := fmt.Sprintf("-- Migration: %s\n-- Created: %s\n\n-- Write your UP migration SQL here\n",
		safeName, time.Now().Format("2006-01-02 15:04:05"))
	downContent := fmt.Sprintf("-- Migration: %s (rollback)\n-- Created: %s\n\n-- Write your DOWN migration SQL here\n",
		safeName, time.Now().Format("2006-01-02 15:04:05"))

	for _, pair := range []struct{ path, content string }{
		{base + ".up.sql", upContent},
		{base + ".down.sql", downContent},
	} {
		if err := os.WriteFile(pair.path, []byte(pair.content), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", pair.path, err)
		}
		ui.PrintFileCreated(pair.path)
	}

	return nil
}

// Validate checks that every .up.sql file has a matching .down.sql and vice versa.
func (r *Runner) Validate() error {
	entries, err := os.ReadDir(r.migrationsDir)
	if os.IsNotExist(err) {
		ui.PrintHint("Migrations directory does not exist yet.")
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading migrations directory: %w", err)
	}

	ups := map[string]bool{}
	downs := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		switch {
		case strings.HasSuffix(n, ".up.sql"):
			ups[strings.TrimSuffix(n, ".up.sql")] = true
		case strings.HasSuffix(n, ".down.sql"):
			downs[strings.TrimSuffix(n, ".down.sql")] = true
		}
	}

	var problems []string
	for base := range ups {
		if !downs[base] {
			problems = append(problems, "missing .down.sql for: "+base)
		}
	}
	for base := range downs {
		if !ups[base] {
			problems = append(problems, "missing .up.sql for: "+base)
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("validation failed:\n  %s", strings.Join(problems, "\n  "))
	}

	ui.PrintDone(fmt.Sprintf("All migration files valid (%d pair(s)).", len(ups)))
	return nil
}

// newMigrate creates a configured *gmigrate.Migrate instance.
func (r *Runner) newMigrate() (*gmigrate.Migrate, func(), error) {
	sourceURL := "file://" + filepath.ToSlash(r.migrationsDir)

	db, err := sql.Open(driverName(r.cfg.Stack.Database), r.dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("opening database: %w", err)
	}

	var drv database.Driver
	switch r.cfg.Stack.Database {
	case config.DatabasePostgres:
		drv, err = gpostgres.WithInstance(db, &gpostgres.Config{})
	case config.DatabaseMySQL:
		drv, err = gmysql.WithInstance(db, &gmysql.Config{})
	default:
		_ = db.Close()
		return nil, nil, fmt.Errorf("unsupported database driver: %s", r.cfg.Stack.Database)
	}
	if err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("creating %s driver: %w", r.cfg.Stack.Database, err)
	}

	m, err := gmigrate.NewWithDatabaseInstance(sourceURL, r.cfg.Stack.Database, drv)
	if err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("creating migrate instance: %w", err)
	}

	cleanup := func() {
		srcErr, dbErr := m.Close()
		_, _ = srcErr, dbErr
		_ = db.Close()
	}

	return m, cleanup, nil
}

// driverName maps config database name to Go sql driver name.
func driverName(database string) string {
	if database == config.DatabaseMySQL {
		return "mysql"
	}
	return "postgres"
}

// sanitizeMigrationName converts a name to snake_case safe for filenames.
func sanitizeMigrationName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r == ' ' || r == '-':
			b.WriteRune('_')
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_':
			b.WriteRune(r)
		}
	}
	return b.String()
}
