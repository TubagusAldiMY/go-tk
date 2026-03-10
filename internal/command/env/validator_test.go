package env

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestValidate_AllPresent(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "DB_HOST=\nDB_PORT=\nJWT_SECRET=required-value\n")
	writeFile(t, dir, ".env", "DB_HOST=localhost\nDB_PORT=5432\nJWT_SECRET=supersecret\n")

	result, err := Validate(filepath.Join(dir, ".env"), filepath.Join(dir, ".env.example"))
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsOK() {
		t.Errorf("expected OK, got missing=%v empty=%v", result.Missing, result.Empty)
	}
}

func TestValidate_MissingKey(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "DB_HOST=\nJWT_SECRET=required\n")
	writeFile(t, dir, ".env", "DB_HOST=localhost\n")

	result, err := Validate(filepath.Join(dir, ".env"), filepath.Join(dir, ".env.example"))
	if err != nil {
		t.Fatal(err)
	}
	if result.IsOK() {
		t.Error("expected not OK due to missing JWT_SECRET")
	}
	if len(result.Missing) != 1 || result.Missing[0] != "JWT_SECRET" {
		t.Errorf("Missing = %v, want [JWT_SECRET]", result.Missing)
	}
}

func TestValidate_MissingEnvFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "KEY=value\n")
	// No .env file

	result, err := Validate(filepath.Join(dir, ".env"), filepath.Join(dir, ".env.example"))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Missing) == 0 {
		t.Error("expected all keys to be missing when .env doesn't exist")
	}
}

func TestSync(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "EXISTING=\nNEW_KEY=\n")
	writeFile(t, dir, ".env", "EXISTING=value\n")

	added, err := Sync(filepath.Join(dir, ".env"), filepath.Join(dir, ".env.example"))
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != 1 || added[0] != "NEW_KEY" {
		t.Errorf("added = %v, want [NEW_KEY]", added)
	}

	// Verify EXISTING is not duplicated
	content, _ := os.ReadFile(filepath.Join(dir, ".env"))
	count := 0
	for _, line := range splitLines(string(content)) {
		if line == "EXISTING=value" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("EXISTING appeared %d times, want 1", count)
	}
}

func TestGenerateExample(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env", "DB_HOST=localhost\nJWT_SECRET=supersecret123\nAPP_ENV=development\n")

	examplePath := filepath.Join(dir, ".env.example")
	if err := GenerateExample(filepath.Join(dir, ".env"), examplePath); err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(examplePath)
	s := string(content)

	if contains(s, "localhost") {
		t.Error(".env.example should not contain actual values")
	}
	if contains(s, "supersecret123") {
		t.Error(".env.example should not contain secret values")
	}
	if !contains(s, "DB_HOST=") {
		t.Error(".env.example should contain DB_HOST key")
	}
}

func TestMaskValue(t *testing.T) {
	cases := []struct {
		key  string
		val  string
		safe bool // true = value should appear, false = should be masked
	}{
		{"APP_NAME", "myapp", true},
		{"JWT_SECRET", "supersecret", false},
		{"DB_PASSWORD", "mypassword", false},
		{"DATABASE_URL", "postgres://user:pass@host/db", false},
		{"PORT", "8080", true},
	}
	for _, c := range cases {
		got := maskValue(c.key, c.val)
		if c.safe && got != c.val {
			t.Errorf("maskValue(%q, %q): expected unchanged %q, got %q", c.key, c.val, c.val, got)
		}
		if !c.safe && got == c.val {
			t.Errorf("maskValue(%q, %q): expected masked value, got original", c.key, c.val)
		}
	}
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
