package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TubagusAldiMY/go-tk/internal/config"
)

func TestRunnerCreate(t *testing.T) {
	dir := t.TempDir()
	migrationsDir := filepath.Join(dir, "migrations")

	r := &Runner{
		cfg:           &config.Config{},
		migrationsDir: migrationsDir,
	}

	if err := r.Create("add_users"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Should have created the migrations directory.
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 files, got %d", len(entries))
	}

	var hasUp, hasDown bool
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, "_add_users.up.sql") {
			hasUp = true
		}
		if strings.HasSuffix(name, "_add_users.down.sql") {
			hasDown = true
		}
	}
	if !hasUp {
		t.Error("missing .up.sql file")
	}
	if !hasDown {
		t.Error("missing .down.sql file")
	}
}

func TestRunnerCreateIdempotent(t *testing.T) {
	dir := t.TempDir()
	migrationsDir := filepath.Join(dir, "migrations")

	r := &Runner{
		cfg:           &config.Config{},
		migrationsDir: migrationsDir,
	}

	// Create twice — should not error.
	if err := r.Create("init"); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}
	if err := r.Create("init"); err != nil {
		t.Fatalf("second Create failed: %v", err)
	}
}

func TestRunnerValidateValid(t *testing.T) {
	dir := t.TempDir()
	pairs := []string{
		"20260101_create_users.up.sql",
		"20260101_create_users.down.sql",
		"20260102_create_orders.up.sql",
		"20260102_create_orders.down.sql",
	}
	for _, f := range pairs {
		os.WriteFile(filepath.Join(dir, f), []byte("-- SQL"), 0o644)
	}

	r := &Runner{cfg: &config.Config{}, migrationsDir: dir}
	if err := r.Validate(); err != nil {
		t.Errorf("Validate failed on valid pairs: %v", err)
	}
}

func TestRunnerValidateMissingDown(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "20260101_orphan.up.sql"), []byte("-- SQL"), 0o644)

	r := &Runner{cfg: &config.Config{}, migrationsDir: dir}
	err := r.Validate()
	if err == nil {
		t.Error("Validate should fail for missing .down.sql")
	}
	if !strings.Contains(err.Error(), "missing .down.sql") {
		t.Errorf("error = %q, want mention of missing .down.sql", err.Error())
	}
}

func TestRunnerValidateMissingUp(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "20260101_orphan.down.sql"), []byte("-- SQL"), 0o644)

	r := &Runner{cfg: &config.Config{}, migrationsDir: dir}
	err := r.Validate()
	if err == nil {
		t.Error("Validate should fail for missing .up.sql")
	}
	if !strings.Contains(err.Error(), "missing .up.sql") {
		t.Errorf("error = %q, want mention of missing .up.sql", err.Error())
	}
}

func TestRunnerValidateNonexistentDir(t *testing.T) {
	r := &Runner{cfg: &config.Config{}, migrationsDir: "/nonexistent/path"}
	if err := r.Validate(); err != nil {
		t.Errorf("Validate should succeed for nonexistent dir, got: %v", err)
	}
}

func TestDriverName(t *testing.T) {
	if got := driverName(config.DatabaseMySQL); got != "mysql" {
		t.Errorf("driverName(mysql) = %q", got)
	}
	if got := driverName(config.DatabasePostgres); got != "postgres" {
		t.Errorf("driverName(postgres) = %q", got)
	}
}
