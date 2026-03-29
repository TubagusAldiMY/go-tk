package ui

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout runs fn and returns whatever was written to os.Stdout.
func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestQuietSuppressesInfoOutput(t *testing.T) {
	Quiet = true
	defer func() { Quiet = false }()

	fns := []struct {
		name string
		fn   func()
	}{
		{"PrintSection", func() { PrintSection("test") }},
		{"PrintStep", func() { PrintStep(1, 3, "step") }},
		{"PrintFileCreated", func() { PrintFileCreated("/tmp/file.go") }},
		{"PrintFileSkipped", func() { PrintFileSkipped("/tmp/file.go") }},
		{"PrintDryRun", func() { PrintDryRun("/tmp/file.go") }},
		{"PrintHint", func() { PrintHint("some hint") }},
		{"PrintBanner", func() { PrintBanner() }},
	}
	for _, tt := range fns {
		t.Run(tt.name, func(t *testing.T) {
			out := captureStdout(tt.fn)
			if out != "" {
				t.Errorf("%s should produce no output when Quiet=true, got %q", tt.name, out)
			}
		})
	}
}

func TestQuietDoesNotSuppressErrorAndDone(t *testing.T) {
	Quiet = true
	defer func() { Quiet = false }()

	errOut := captureStdout(func() { PrintError("fail") })
	if errOut == "" {
		t.Error("PrintError should produce output even when Quiet=true")
	}

	doneOut := captureStdout(func() { PrintDone("success") })
	if doneOut == "" {
		t.Error("PrintDone should produce output even when Quiet=true")
	}
}

func TestPrintFunctionsProduceOutputWhenNotQuiet(t *testing.T) {
	Quiet = false

	fns := []struct {
		name string
		fn   func()
		want string
	}{
		{"PrintError", func() { PrintError("broken") }, "broken"},
		{"PrintDone", func() { PrintDone("finished") }, "finished"},
		{"PrintSection", func() { PrintSection("header") }, "header"},
		{"PrintHint", func() { PrintHint("tip") }, "tip"},
		{"PrintFileCreated", func() { PrintFileCreated("a.go") }, "a.go"},
		{"PrintFileSkipped", func() { PrintFileSkipped("a.go") }, "a.go"},
		{"PrintDryRun", func() { PrintDryRun("a.go") }, "a.go"},
	}
	for _, tt := range fns {
		t.Run(tt.name, func(t *testing.T) {
			out := captureStdout(tt.fn)
			if !strings.Contains(out, tt.want) {
				t.Errorf("%s output %q does not contain %q", tt.name, out, tt.want)
			}
		})
	}
}

func TestBannerContainsBrandName(t *testing.T) {
	b := Banner()
	if !strings.Contains(b, "go-tk") {
		t.Errorf("Banner() = %q, want it to contain %q", b, "go-tk")
	}
}

func TestMsgFunctions(t *testing.T) {
	if s := SuccessMsg("ok"); !strings.Contains(s, "ok") {
		t.Errorf("SuccessMsg = %q, missing content", s)
	}
	if s := ErrorMsg("fail"); !strings.Contains(s, "fail") {
		t.Errorf("ErrorMsg = %q, missing content", s)
	}
	if s := InfoMsg("info"); !strings.Contains(s, "info") {
		t.Errorf("InfoMsg = %q, missing content", s)
	}
}
