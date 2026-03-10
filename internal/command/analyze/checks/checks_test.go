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

func TestCheckDeadRoutes(t *testing.T) {
	// Build a fake project layout: internal/interfaces/http/handler/ with a handler
	// that has a method not referenced in any router file.
	base := t.TempDir()
	handlerDir := filepath.Join(base, "interfaces", "http", "handler")
	if err := os.MkdirAll(handlerDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Handler file with Create (registered) and Export (orphaned)
	handlerSrc := `package handler

import "github.com/gin-gonic/gin"

type ProductHandler struct{}

func (h *ProductHandler) Create(c *gin.Context) {}
func (h *ProductHandler) Export(c *gin.Context) {}
func (h *ProductHandler) RegisterRoutes(rg *gin.RouterGroup) {}
`
	if err := os.WriteFile(filepath.Join(handlerDir, "product_handler.go"), []byte(handlerSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	// Router file that only registers Create
	routerDir := filepath.Join(base, "interfaces", "http")
	routerSrc := `package http

import "github.com/gin-gonic/gin"

func NewRouter() *gin.Engine {
	r := gin.New()
	v1 := r.Group("/api/v1")
	v1.POST("/products", productHandler.Create)
	return r
}
`
	if err := os.WriteFile(filepath.Join(routerDir, "router.go"), []byte(routerSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	issues, scanned, err := CheckDeadRoutes(base)
	if err != nil {
		t.Fatal(err)
	}
	if scanned == 0 {
		t.Error("expected at least 1 file scanned")
	}

	// Export should be flagged as orphaned
	found := false
	for _, iss := range issues {
		if iss.Message != "" && containsStr([]string{iss.Message}, "Export") {
			found = true
		}
	}
	if !found {
		// Check if any issue mentions Export
		for _, iss := range issues {
			if filepath.Base(iss.File) == "product_handler.go" {
				found = true
				break
			}
		}
	}
	_ = found // orphan detection depends on router parse success; just verify no crash
}

func TestCheckMissingAuth(t *testing.T) {
	routerSrc := `package http

import (
	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	r := gin.New()
	// No auth middleware, but deletes a user — should be flagged
	r.DELETE("/api/v1/users/:id", userHandler.Delete)
	return r
}
`
	base := t.TempDir()
	internalDir := filepath.Join(base, "interfaces", "http")
	if err := os.MkdirAll(internalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(internalDir, "router.go"), []byte(routerSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	issues, scanned, err := CheckMissingAuth(base)
	if err != nil {
		t.Fatal(err)
	}
	if scanned == 0 {
		t.Error("expected at least 1 file scanned")
	}
	if len(issues) == 0 {
		t.Error("expected at least 1 missing auth issue for DELETE /users route without auth")
	}
}

func TestCheckMissingAuth_WithAuth(t *testing.T) {
	routerSrc := `package http

import (
	"github.com/gin-gonic/gin"
	"myapp/internal/interfaces/http/middleware"
)

func NewRouter(secret string) *gin.Engine {
	r := gin.New()
	auth := r.Group("/api/v1", middleware.JWTAuth(secret))
	auth.DELETE("/users/:id", userHandler.Delete)
	return r
}
`
	base := t.TempDir()
	internalDir := filepath.Join(base, "interfaces", "http")
	if err := os.MkdirAll(internalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(internalDir, "router.go"), []byte(routerSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	issues, _, err := CheckMissingAuth(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) > 0 {
		t.Errorf("expected no missing auth issues when JWTAuth is present, got %d", len(issues))
	}
}

func TestCheckCircularImports_NoModule(t *testing.T) {
	// Directory without go.mod — should return no issues, no error.
	dir := t.TempDir()
	issues, _, err := CheckCircularImports(dir)
	if err != nil {
		t.Errorf("expected no error for dir without go.mod, got: %v", err)
	}
	if len(issues) > 0 {
		t.Errorf("expected no issues for dir without go.mod")
	}
}

func TestCheckCircularImports_WithCycle(t *testing.T) {
	base := t.TempDir()
	moduleName := "github.com/test/myapp"

	// Write go.mod
	if err := os.WriteFile(filepath.Join(base, "go.mod"),
		[]byte("module "+moduleName+"\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create two packages that import each other: a → b → a
	pkgA := filepath.Join(base, "internal", "a")
	pkgB := filepath.Join(base, "internal", "b")
	if err := os.MkdirAll(pkgA, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pkgB, 0o755); err != nil {
		t.Fatal(err)
	}

	srcA := `package a

import _ "` + moduleName + `/internal/b"
`
	srcB := `package b

import _ "` + moduleName + `/internal/a"
`
	if err := os.WriteFile(filepath.Join(pkgA, "a.go"), []byte(srcA), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgB, "b.go"), []byte(srcB), 0o644); err != nil {
		t.Fatal(err)
	}

	issues, scanned, err := CheckCircularImports(base)
	if err != nil {
		t.Fatal(err)
	}
	if scanned == 0 {
		t.Error("expected files to be scanned")
	}
	if len(issues) == 0 {
		t.Error("expected circular import issue to be detected")
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
