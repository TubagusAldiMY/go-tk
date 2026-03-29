package analyze

import (
	"fmt"
	"html/template"
	"os"
	"strings"
	"time"

	"github.com/TubagusAldiMY/go-tk/internal/command/analyze/types"
)

var htmlFuncMap = template.FuncMap{
	"lower": strings.ToLower,
}

// htmlReportData is the template data for the HTML report.
type htmlReportData struct {
	ProjectName  string
	Timestamp    string
	HealthScore  int
	Grade        string
	GradeClass   string
	FilesScanned int
	TotalIssues  int
	Critical     int
	High         int
	Medium       int
	Low          int
	Issues       []types.Issue
}

// outputHTML writes a self-contained HTML report file.
func outputHTML(result *types.AnalysisResult, projectName string) (string, error) {
	summary := countSeverities(result.Issues)

	gradeClass := "grade-a"
	switch {
	case result.HealthScore < 40:
		gradeClass = "grade-f"
	case result.HealthScore < 60:
		gradeClass = "grade-d"
	case result.HealthScore < 75:
		gradeClass = "grade-c"
	case result.HealthScore < 90:
		gradeClass = "grade-b"
	}

	data := htmlReportData{
		ProjectName:  projectName,
		Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
		HealthScore:  result.HealthScore,
		Grade:        types.ScoreGrade(result.HealthScore),
		GradeClass:   gradeClass,
		FilesScanned: result.FilesScanned,
		TotalIssues:  len(result.Issues),
		Critical:     summary.Critical,
		High:         summary.High,
		Medium:       summary.Medium,
		Low:          summary.Low,
		Issues:       result.Issues,
	}

	tmpl, err := template.New("report").Funcs(htmlFuncMap).Parse(htmlTemplate)
	if err != nil {
		return "", fmt.Errorf("parsing HTML template: %w", err)
	}

	filename := fmt.Sprintf("analyze-report-%s.html", time.Now().Format("20060102-150405"))
	f, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("creating report file: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return "", fmt.Errorf("rendering HTML report: %w", err)
	}

	return filename, nil
}

func countSeverities(issues []types.Issue) IssueSummary {
	s := IssueSummary{Total: len(issues)}
	for _, i := range issues {
		switch i.Severity {
		case types.SeverityCritical:
			s.Critical++
		case types.SeverityHigh:
			s.High++
		case types.SeverityMedium:
			s.Medium++
		case types.SeverityLow:
			s.Low++
		case types.SeverityInfo:
			s.Info++
		}
	}
	return s
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>go-tk analyze — {{.ProjectName}}</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0f172a; color: #e2e8f0; padding: 2rem; }
  .container { max-width: 900px; margin: 0 auto; }
  h1 { font-size: 1.5rem; color: #a78bfa; margin-bottom: .25rem; }
  .meta { color: #64748b; font-size: .85rem; margin-bottom: 2rem; }
  .dashboard { display: grid; grid-template-columns: 1fr 1fr; gap: 1.5rem; margin-bottom: 2rem; }
  .card { background: #1e293b; border-radius: .75rem; padding: 1.5rem; }
  .score-card { text-align: center; }
  .score { font-size: 3.5rem; font-weight: 700; }
  .grade { display: inline-block; font-size: 1.25rem; font-weight: 700; padding: .25rem .75rem; border-radius: .5rem; margin-top: .5rem; }
  .grade-a { background: #166534; color: #bbf7d0; }
  .grade-b { background: #854d0e; color: #fef08a; }
  .grade-c { background: #92400e; color: #fed7aa; }
  .grade-d { background: #7c2d12; color: #fecaca; }
  .grade-f { background: #7f1d1d; color: #fecaca; }
  .bar { height: 8px; background: #334155; border-radius: 4px; margin-top: 1rem; overflow: hidden; }
  .bar-fill { height: 100%; border-radius: 4px; }
  .summary-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: .75rem; }
  .summary-item { text-align: center; }
  .summary-count { font-size: 1.75rem; font-weight: 700; }
  .summary-label { font-size: .75rem; text-transform: uppercase; color: #64748b; }
  .critical { color: #f87171; }
  .high { color: #fb923c; }
  .medium { color: #fbbf24; }
  .low { color: #60a5fa; }
  .issues { margin-top: 1rem; }
  .issue { background: #1e293b; border-radius: .5rem; padding: 1rem; margin-bottom: .5rem; display: flex; gap: 1rem; align-items: flex-start; }
  .badge { font-size: .7rem; font-weight: 700; padding: .15rem .5rem; border-radius: .25rem; text-transform: uppercase; white-space: nowrap; }
  .badge-critical { background: #7f1d1d; color: #fca5a5; }
  .badge-high { background: #7c2d12; color: #fdba74; }
  .badge-medium { background: #78350f; color: #fde68a; }
  .badge-low { background: #1e3a5f; color: #93c5fd; }
  .issue-body { flex: 1; }
  .issue-msg { font-size: .9rem; }
  .issue-loc { font-size: .8rem; color: #64748b; margin-top: .25rem; font-family: monospace; }
  .empty { text-align: center; color: #22c55e; padding: 3rem; font-size: 1.1rem; }
  .footer { text-align: center; color: #475569; font-size: .75rem; margin-top: 2rem; }
</style>
</head>
<body>
<div class="container">
  <h1>go-tk analyze</h1>
  <div class="meta">Project: <strong>{{.ProjectName}}</strong> &middot; {{.Timestamp}} &middot; {{.FilesScanned}} files scanned</div>

  <div class="dashboard">
    <div class="card score-card">
      <div class="score">{{.HealthScore}}</div>
      <div>/100</div>
      <span class="grade {{.GradeClass}}">Grade {{.Grade}}</span>
      <div class="bar"><div class="bar-fill {{.GradeClass}}" style="width:{{.HealthScore}}%"></div></div>
    </div>
    <div class="card">
      <div class="summary-grid">
        <div class="summary-item"><div class="summary-count critical">{{.Critical}}</div><div class="summary-label">Critical</div></div>
        <div class="summary-item"><div class="summary-count high">{{.High}}</div><div class="summary-label">High</div></div>
        <div class="summary-item"><div class="summary-count medium">{{.Medium}}</div><div class="summary-label">Medium</div></div>
        <div class="summary-item"><div class="summary-count low">{{.Low}}</div><div class="summary-label">Low</div></div>
      </div>
      <div style="text-align:center; margin-top:1.5rem; color:#94a3b8;">{{.TotalIssues}} total issue(s)</div>
    </div>
  </div>

  {{if .Issues}}
  <h2 style="font-size:1.1rem; margin-bottom:1rem;">Issues</h2>
  <div class="issues">
    {{range .Issues}}
    <div class="issue">
      <span class="badge badge-{{.Severity.String | lower}}">{{.Severity}}</span>
      <div class="issue-body">
        <div class="issue-msg">{{.Message}}</div>
        <div class="issue-loc">{{.File}}{{if .Line}}:{{.Line}}{{end}} &middot; {{.Kind}}</div>
      </div>
    </div>
    {{end}}
  </div>
  {{else}}
  <div class="empty">No issues found — your code is clean!</div>
  {{end}}

  <div class="footer">Generated by go-tk analyze &middot; https://github.com/TubagusAldiMY/go-tk</div>
</div>
</body>
</html>`
