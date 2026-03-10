package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseStructFromFile(t *testing.T) {
	src := `package entity

import "time"

type Product struct {
	ID        uint      ` + "`gorm:\"primaryKey\" json:\"id\"`" + `
	Name      string    ` + "`gorm:\"column:name\" json:\"name\"`" + `
	Price     float64   ` + "`gorm:\"column:price\" json:\"price\"`" + `
	CreatedAt time.Time
}
`
	dir := t.TempDir()
	file := filepath.Join(dir, "product.go")
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	ps, err := ParseStructFromFile(file, "Product")
	if err != nil {
		t.Fatalf("ParseStructFromFile failed: %v", err)
	}

	if ps.Name != "Product" {
		t.Errorf("Name = %q, want Product", ps.Name)
	}
	if len(ps.Fields) != 4 {
		t.Errorf("len(Fields) = %d, want 4", len(ps.Fields))
	}

	// Check first field
	if ps.Fields[0].Name != "ID" {
		t.Errorf("Fields[0].Name = %q, want ID", ps.Fields[0].Name)
	}
	if ps.Fields[0].Type != "uint" {
		t.Errorf("Fields[0].Type = %q, want uint", ps.Fields[0].Type)
	}
}

func TestParseStructNotFound(t *testing.T) {
	src := `package entity
type Order struct { ID uint }
`
	dir := t.TempDir()
	file := filepath.Join(dir, "order.go")
	_ = os.WriteFile(file, []byte(src), 0o644)

	_, err := ParseStructFromFile(file, "NonExistent")
	if err == nil {
		t.Fatal("expected ErrStructNotFound")
	}
}

func TestParseStructTags(t *testing.T) {
	src := `package entity
type User struct {
	Email string ` + "`json:\"email\" validate:\"required,email\"`" + `
}
`
	dir := t.TempDir()
	file := filepath.Join(dir, "user.go")
	_ = os.WriteFile(file, []byte(src), 0o644)

	ps, err := ParseStructFromFile(file, "User")
	if err != nil {
		t.Fatal(err)
	}

	if ps.Fields[0].Tags["json"] != "email" {
		t.Errorf("json tag = %q, want %q", ps.Fields[0].Tags["json"], "email")
	}
	if ps.Fields[0].Tags["validate"] != "required,email" {
		t.Errorf("validate tag = %q", ps.Fields[0].Tags["validate"])
	}
}
