package generator

import (
	"embed"
	"testing"
)

// testFS is a minimal in-memory FS for testing the engine.
// We use a real embed.FS so we can test the full Render path.
// The test templates live in testdata/ within this package.

//go:embed testdata
var testFS embed.FS

func TestEngineRender(t *testing.T) {
	engine := NewEngine(testFS, "testdata")

	data := map[string]string{"Name": "World"}
	got, err := engine.Render("hello.tmpl", data)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	want := "Hello, World!"
	if string(got) != want {
		t.Errorf("Render = %q, want %q", got, want)
	}
}

func TestEngineRenderString(t *testing.T) {
	engine := NewEngine(testFS, "testdata")

	got, err := engine.RenderString("inline", "value={{.V}}", map[string]string{"V": "42"})
	if err != nil {
		t.Fatalf("RenderString failed: %v", err)
	}
	if string(got) != "value=42" {
		t.Errorf("RenderString = %q, want %q", got, "value=42")
	}
}

func TestEngineListTemplates(t *testing.T) {
	engine := NewEngine(testFS, "testdata")

	paths, err := engine.ListTemplates("")
	if err != nil {
		t.Fatalf("ListTemplates failed: %v", err)
	}
	if len(paths) == 0 {
		t.Error("expected at least one template")
	}

	found := false
	for _, p := range paths {
		if p == "hello.tmpl" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("hello.tmpl not found in %v", paths)
	}
}
