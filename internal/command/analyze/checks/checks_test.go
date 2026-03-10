package checks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckUnhandledErrors(t *testing.T) {
	src := `package main

import "os"

func main() {
	_ = os.Remove("/tmp/test")
	_ = os.WriteFile("/tmp/x", nil, 0644)
}
`
	dir := writeTempFile(t, "unhandled.go", src)

	issues, scanned, err := CheckUnhandledErrors(dir)
	if err != nil {
		t.Fatal(err)
	}
	if scanned == 0 {
		t.Error("expected at least 1 file scanned")
	}
	if len(issues) == 0 {
		t.Error("expected at least 1 unhandled error issue")
	}
}

func TestCheckMissingValidation(t *testing.T) {
	src := `package handler

import "github.com/gin-gonic/gin"

type Handler struct{}

func (h *Handler) Create(c *gin.Context) {
	var req struct{ Name string }
	if err := c.ShouldBindJSON(&req); err != nil {
		return
	}
	// No validator.Struct call — should be flagged
}
`
	dir := writeTempFile(t, "product_handler.go", src)

	issues, _, err := CheckMissingValidation(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) == 0 {
		t.Error("expected missing validation issue")
	}
}

func TestCheckNPlusOne(t *testing.T) {
	src := `package usecase

import "gorm.io/gorm"

func loadAll(db *gorm.DB, ids []uint) {
	for _, id := range ids {
		var item struct{ ID uint }
		db.First(&item, id)  // N+1!
	}
}
`
	dir := writeTempFile(t, "usecase.go", src)

	issues, _, err := CheckNPlusOne(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) == 0 {
		t.Error("expected N+1 issue")
	}
}

func TestCheckHardcodedValues(t *testing.T) {
	src := `package main

func main() {
	password := "supersecret123"
	_ = password
}
`
	dir := writeTempFile(t, "main.go", src)

	issues, _, err := CheckHardcodedValues(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) == 0 {
		t.Error("expected hardcoded credential issue")
	}
}

// writeTempFile creates a temp dir with one Go source file and returns the dir path.
func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}
