package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseRoutesFromFile(t *testing.T) {
	src := `package http

import "github.com/gin-gonic/gin"

func NewRouter() *gin.Engine {
	r := gin.New()
	r.GET("/health", healthHandler)
	r.POST("/api/v1/users", userHandler.Create)
	r.GET("/api/v1/users/:id", userHandler.GetByID)
	r.PUT("/api/v1/users/:id", userHandler.Update)
	r.DELETE("/api/v1/users/:id", userHandler.Delete)
	return r
}
`
	dir := t.TempDir()
	file := filepath.Join(dir, "router.go")
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	routes, err := ParseRoutesFromFile(file)
	if err != nil {
		t.Fatalf("ParseRoutesFromFile failed: %v", err)
	}

	if len(routes) != 5 {
		t.Fatalf("len(routes) = %d, want 5", len(routes))
	}

	// Verify methods
	methods := map[string]bool{}
	for _, r := range routes {
		methods[r.Method] = true
	}
	for _, m := range []string{"GET", "POST", "PUT", "DELETE"} {
		if !methods[m] {
			t.Errorf("method %s not found in routes", m)
		}
	}
}

func TestParseRoutesEmpty(t *testing.T) {
	src := `package http
func setup() {}
`
	dir := t.TempDir()
	file := filepath.Join(dir, "empty.go")
	_ = os.WriteFile(file, []byte(src), 0o644)

	routes, err := ParseRoutesFromFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 0 {
		t.Errorf("expected 0 routes, got %d", len(routes))
	}
}
