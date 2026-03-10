package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseMigrationFilename(t *testing.T) {
	tests := []struct {
		input       string
		wantVersion uint
		wantName    string
		wantOK      bool
	}{
		{"20260310120000_create_users.up.sql", 20260310120000, "create_users", true},
		{"001_init.up.sql", 1, "init", true},
		{"noversion.up.sql", 0, "", false},
		{"invalid.txt", 0, "", false},
	}
	for _, tt := range tests {
		v, name, ok := parseMigrationFilename(tt.input)
		if ok != tt.wantOK {
			t.Errorf("parseMigrationFilename(%q) ok=%v, want %v", tt.input, ok, tt.wantOK)
			continue
		}
		if ok && v != tt.wantVersion {
			t.Errorf("parseMigrationFilename(%q) version=%d, want %d", tt.input, v, tt.wantVersion)
		}
		if ok && name != tt.wantName {
			t.Errorf("parseMigrationFilename(%q) name=%q, want %q", tt.input, name, tt.wantName)
		}
	}
}

func TestGetStatus(t *testing.T) {
	dir := t.TempDir()

	// Create some migration files
	files := []string{
		"20260101000000_create_users.up.sql",
		"20260101000000_create_users.down.sql",
		"20260102000000_create_products.up.sql",
		"20260102000000_create_products.down.sql",
		"20260103000000_create_orders.up.sql",
		"20260103000000_create_orders.down.sql",
	}
	for _, f := range files {
		_ = os.WriteFile(filepath.Join(dir, f), []byte("-- SQL"), 0o644)
	}

	// First two are applied (version <= 20260102000000)
	statuses, err := GetStatus(dir, 20260102000000, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 3 {
		t.Fatalf("expected 3 statuses, got %d", len(statuses))
	}
	if !statuses[0].Applied {
		t.Error("first migration should be applied")
	}
	if !statuses[1].Applied {
		t.Error("second migration should be applied")
	}
	if statuses[2].Applied {
		t.Error("third migration should be pending")
	}
}

func TestGetStatusMissingDir(t *testing.T) {
	statuses, err := GetStatus("/nonexistent/path", 0, false)
	if err != nil {
		t.Fatalf("expected nil error for missing dir, got: %v", err)
	}
	if statuses != nil {
		t.Error("expected nil statuses for missing dir")
	}
}

func TestSanitizeMigrationName(t *testing.T) {
	cases := []struct{ input, want string }{
		{"create users table", "create_users_table"},
		{"add-index-products", "add_index_products"},
		{"CreateOrders", "createorders"},
		{"hello_world", "hello_world"},
	}
	for _, c := range cases {
		got := sanitizeMigrationName(c.input)
		if got != c.want {
			t.Errorf("sanitizeMigrationName(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestSanitizeForDisplay(t *testing.T) {
	dsn := "postgres://user:password@localhost:5432/mydb?sslmode=disable"
	got := sanitizeForDisplay(dsn)
	if got != "localhost:5432/mydb?sslmode=disable" {
		t.Errorf("sanitizeForDisplay = %q", got)
	}

	// No @ sign — return as-is
	plain := "no-at-sign"
	if sanitizeForDisplay(plain) != plain {
		t.Error("sanitizeForDisplay should return unchanged when no @")
	}
}
