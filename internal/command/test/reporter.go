package test

import (
	"fmt"
	"html/template"
	"os"
	"strings"
	"time"

	"github.com/TubagusAldiMY/go-tk/internal/ui"
)

// PrintTerminalReport prints a human-readable test report to stdout.
func PrintTerminalReport(results []TestResult) {
	ui.PrintSection("API Test Results")

	maxName := 0
	for _, r := range results {
		if len(r.Name) > maxName {
			maxName = len(r.Name)
		}
	}

	for _, r := range results {
		pad := strings.Repeat(" ", maxName-len(r.Name))
		duration := fmt.Sprintf("%4dms", r.Duration.Milliseconds())

		if r.Error != "" {
			fmt.Printf("  %s  %s%s  %s  %s\n",
				ui.StyleError.Render("✗"),
				r.Name, pad,
				ui.StyleMuted.Render(duration),
				ui.StyleError.Render("ERROR: "+r.Error),
			)
		} else if r.Pass {
			status := ui.StyleSuccess.Render(fmt.Sprintf("%d", r.StatusCode))
			fmt.Printf("  %s  %s%s  %s  %s\n",
				ui.StyleSuccess.Render("✓"),
				r.Name, pad,
				ui.StyleMuted.Render(duration),
				status,
			)
		} else {
			status := ui.StyleError.Render(fmt.Sprintf("%d", r.StatusCode))
			fmt.Printf("  %s  %s%s  %s  %s\n",
				ui.StyleError.Render("✗"),
				r.Name, pad,
				ui.StyleMuted.Render(duration),
				status,
			)
		}
	}

	passed, failed := Summary(results)
	fmt.Println(strings.Repeat("─", 55))
	summary := fmt.Sprintf("%d passed · %d failed · %d total", passed, failed, len(results))
	if failed > 0 {
		fmt.Printf("  %s\n\n", ui.StyleError.Render(summary))
	} else {
		fmt.Printf("  %s\n\n", ui.StyleSuccess.Render(summary))
	}
}

// WriteHTMLReport writes a styled HTML report to outputPath.
func WriteHTMLReport(results []TestResult, outputPath string) error {
	type reportData struct {
		GeneratedAt string
		Results     []TestResult
		Passed      int
		Failed      int
		Total       int
	}

	passed, failed := Summary(results)
	data := reportData{
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		Results:     results,
		Passed:      passed,
		Failed:      failed,
		Total:       len(results),
	}

	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"statusClass": func(r TestResult) string {
			if r.Error != "" || !r.Pass {
				return "fail"
			}
			return "pass"
		},
		"statusIcon": func(r TestResult) string {
			if r.Error != "" || !r.Pass {
				return "✗"
			}
			return "✓"
		},
	}).Parse(htmlReportTemplate)
	if err != nil {
		return fmt.Errorf("parsing report template: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating report file: %w", err)
	}
	defer f.Close()

	return tmpl.Execute(f, data)
}

const htmlReportTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>go-tk API Test Report</title>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; margin: 0; padding: 24px; background: #f9fafb; color: #111827; }
  h1 { font-size: 1.5rem; margin-bottom: 4px; }
  .meta { color: #6b7280; font-size: 0.875rem; margin-bottom: 24px; }
  .summary { display: flex; gap: 16px; margin-bottom: 24px; }
  .badge { padding: 6px 16px; border-radius: 6px; font-weight: 600; font-size: 0.875rem; }
  .badge-pass { background: #dcfce7; color: #15803d; }
  .badge-fail { background: #fee2e2; color: #b91c1c; }
  .badge-total { background: #e0e7ff; color: #3730a3; }
  table { width: 100%%; border-collapse: collapse; background: #fff; border-radius: 8px; overflow: hidden; box-shadow: 0 1px 3px rgba(0,0,0,.1); }
  th { background: #f3f4f6; text-align: left; padding: 10px 16px; font-size: 0.75rem; text-transform: uppercase; letter-spacing: .05em; color: #6b7280; }
  td { padding: 10px 16px; border-top: 1px solid #f3f4f6; font-size: 0.875rem; }
  .pass td:first-child { color: #15803d; }
  .fail td:first-child { color: #b91c1c; }
  .method { font-weight: 600; padding: 2px 8px; border-radius: 4px; background: #e0e7ff; color: #3730a3; font-size: 0.75rem; }
  .code-2xx { color: #15803d; font-weight: 600; }
  .code-4xx { color: #d97706; font-weight: 600; }
  .code-5xx { color: #b91c1c; font-weight: 600; }
  .duration { color: #6b7280; }
  .error-msg { color: #b91c1c; font-size: 0.8rem; }
</style>
</head>
<body>
<h1>go-tk API Test Report</h1>
<p class="meta">Generated: {{.GeneratedAt}}</p>
<div class="summary">
  <span class="badge badge-pass">✓ {{.Passed}} passed</span>
  <span class="badge badge-fail">✗ {{.Failed}} failed</span>
  <span class="badge badge-total">{{.Total}} total</span>
</div>
<table>
  <thead>
    <tr><th></th><th>Method</th><th>Path</th><th>Status</th><th>Duration</th><th>Details</th></tr>
  </thead>
  <tbody>
  {{range .Results}}
    <tr class="{{statusClass .}}">
      <td>{{statusIcon .}}</td>
      <td><span class="method">{{.Method}}</span></td>
      <td>{{.Path}}</td>
      <td>{{if .StatusCode}}<span class="{{if lt .StatusCode 300}}code-2xx{{else if lt .StatusCode 500}}code-4xx{{else}}code-5xx{{end}}">{{.StatusCode}}</span>{{end}}</td>
      <td class="duration">{{.Duration.Milliseconds}}ms</td>
      <td>{{if .Error}}<span class="error-msg">{{.Error}}</span>{{end}}</td>
    </tr>
  {{end}}
  </tbody>
</table>
</body>
</html>`
