package generator

import (
	"testing"
)

func TestFormatGo(t *testing.T) {
	unformatted := []byte(`package main
import "fmt"
func main(){fmt.Println("hi")}`)

	formatted, err := FormatGo(unformatted)
	if err != nil {
		t.Fatalf("FormatGo failed: %v", err)
	}

	// Must produce valid, different output.
	if string(formatted) == string(unformatted) {
		t.Error("FormatGo should change unformatted source")
	}
}

func TestFormatGoInvalid(t *testing.T) {
	// Invalid Go should return original content, not an error that breaks generation.
	invalid := []byte(`this is not go code {{{{`)
	out, err := FormatGo(invalid)
	if err == nil {
		t.Error("expected error for invalid Go")
	}
	// Should return original content unchanged so generation continues.
	if string(out) != string(invalid) {
		t.Error("FormatGo should return original content on parse error")
	}
}

func TestIsGoFile(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"main.go", true},
		{"handler.go", true},
		{"main.go.tmpl", false},
		{"README.md", false},
		{"migration.sql", false},
	}
	for _, c := range cases {
		if IsGoFile(c.path) != c.want {
			t.Errorf("IsGoFile(%q) = %v, want %v", c.path, !c.want, c.want)
		}
	}
}
