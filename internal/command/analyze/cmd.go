// Package analyze implements the "go-tk analyze" command.
//
// This package performs static analysis on Go backend projects to detect:
//  1. Code quality issues (unhandled errors, missing validation)
//  2. Performance problems (N+1 queries)
//  3. Security vulnerabilities (hardcoded credentials, missing auth)
//  4. Architecture violations (circular imports, dead routes)
//
// Analysis Strategy:
//
//	All checks use AST parsing (go/ast) and file system traversal — NO CODE EXECUTION.
//	This means analysis is safe to run on untrusted code and works on broken projects.
//
// Health Score Algorithm:
//
//	Base score: 100
//	Per issue: -penalty based on severity
//	  CRITICAL → -10 points
//	  HIGH     → -5 points
//	  MEDIUM   → -2 points
//	  LOW      → -1 point
//	Floor: 0 (cannot go negative)
//
// Grade Scale:
//
//	90-100 → A (Excellent)
//	75-89  → B (Good)
//	60-74  → C (Fair)
//	40-59  → D (Poor)
//	0-39   → F (Critical)
//
// Output Modes:
//   - Text (default): Human-readable colored output with progress bars
//   - JSON (TODO #5): Machine-readable for CI pipelines
//   - HTML (TODO #5): Web report with charts and filtering
//
// CI Integration:
//
//	Use --fail-under to enforce minimum health score:
//	  go-tk analyze --fail-under=75
//	Exit code 0 if score >= threshold, 1 otherwise.
package analyze

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/TubagusAldiMY/go-tk/internal/command/analyze/checks"
	"github.com/TubagusAldiMY/go-tk/internal/command/analyze/types"
	"github.com/TubagusAldiMY/go-tk/internal/config"
	"github.com/TubagusAldiMY/go-tk/internal/ui"
)

// JSONOutput is the machine-readable output format for CI integration.
// Follows the standard response envelope pattern from AGENTS.md Section 5.4.
type JSONOutput struct {
	Success bool            `json:"success"`
	Data    *JSONOutputData `json:"data"`
	Meta    *JSONOutputMeta `json:"meta"`
}

// JSONOutputData contains the analysis results.
type JSONOutputData struct {
	HealthScore int           `json:"health_score"`
	Grade       string        `json:"grade"`
	Issues      []JSONIssue   `json:"issues"`
	Summary     IssueSummary  `json:"summary"`
}

// JSONIssue is a single issue in JSON format.
type JSONIssue struct {
	Kind     string `json:"kind"`
	Severity string `json:"severity"`
	File     string `json:"file"`
	Line     int    `json:"line,omitempty"`
	Message  string `json:"message"`
}

// IssueSummary contains counts by severity.
type IssueSummary struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Info     int `json:"info"`
	Total    int `json:"total"`
}

// JSONOutputMeta contains metadata about the analysis run.
type JSONOutputMeta struct {
	FilesScanned int       `json:"files_scanned"`
	Timestamp    time.Time `json:"timestamp"`
	ProjectName  string    `json:"project_name"`
	FailUnder    int       `json:"fail_under,omitempty"`
}

// AnalyzeCmd returns the cobra.Command for "go-tk analyze".
func AnalyzeCmd() *cobra.Command {
	var minSeverity string
	var failUnder int
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze project code health (dead routes, N+1, missing validation, ...)",
		Long: `Static analysis of your Go backend project.

Checks performed:
  • Unhandled errors (errors discarded with _)
  • Missing input validation (bind without validate)
  • N+1 query patterns (DB call inside loop)
  • Hardcoded credentials and config values
  • Dead routes & orphaned handlers (routes with no handler / handlers never routed)
  • Missing auth middleware (mutable routes on sensitive paths without auth)
  • Circular imports (import cycles in internal packages)

Outputs a health score (0–100) and a graded report.

Use --fail-under in CI pipelines to enforce a minimum health score:
  go-tk analyze --fail-under=75

Use --output=json for machine-readable output (CI integration):
  go-tk analyze --output=json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnalyze(minSeverity, failUnder, outputFormat)
		},
	}

	cmd.Flags().StringVar(&minSeverity, "min-severity", "low", "Minimum severity to report (critical|high|medium|low|info)")
	cmd.Flags().IntVar(&failUnder, "fail-under", 0, "Exit with code 1 if health score is below this threshold (0 = disabled)")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "Output format: text | json")

	return cmd
}

func runAnalyze(minSeverityStr string, failUnder int, outputFormat string) error {
	// Validate output format
	if outputFormat != "text" && outputFormat != "json" {
		return fmt.Errorf("invalid output format: %s (valid: text, json)", outputFormat)
	}

	cwd, _ := os.Getwd()
	cfg, err := config.Load(cwd)
	if err != nil {
		if outputFormat == "json" {
			return outputJSONError("config not found: run 'go-tk new' first")
		}
		return config.ErrConfigNotFound
	}

	// Skip banner for JSON output
	if outputFormat != "json" {
		fmt.Println()
		fmt.Println(ui.Banner())
		fmt.Println()
		ui.PrintSection("Analyzing project: " + cfg.Project.Name)
	}

	result := &types.AnalysisResult{}

	// Run all checks
	runCheck(result, "Unhandled errors", func() ([]types.Issue, int, error) {
		return checks.CheckUnhandledErrors(cwd + "/internal")
	})

	runCheck(result, "Missing input validation", func() ([]types.Issue, int, error) {
		return checks.CheckMissingValidation(cfg.Paths.Handlers)
	})

	runCheck(result, "N+1 query patterns", func() ([]types.Issue, int, error) {
		return checks.CheckNPlusOne(cwd + "/internal")
	})

	runCheck(result, "Hardcoded values", func() ([]types.Issue, int, error) {
		return checks.CheckHardcodedValues(cwd + "/internal")
	})

	runCheck(result, "Dead routes & orphaned handlers", func() ([]types.Issue, int, error) {
		return checks.CheckDeadRoutes(cwd + "/internal")
	})

	runCheck(result, "Missing auth middleware", func() ([]types.Issue, int, error) {
		return checks.CheckMissingAuth(cwd + "/internal")
	})

	runCheck(result, "Circular imports", func() ([]types.Issue, int, error) {
		return checks.CheckCircularImports(cwd)
	})

	// Compute health score
	result.HealthScore = types.ComputeHealthScore(result.Issues)

	// Output based on format
	if outputFormat == "json" {
		return outputJSON(result, cfg.Project.Name, failUnder)
	}

	// Print text report
	printReport(result, parseSeverity(minSeverityStr))

	// Enforce minimum health score for CI pipelines.
	if failUnder > 0 && result.HealthScore < failUnder {
		return fmt.Errorf("health score %d is below minimum threshold %d", result.HealthScore, failUnder)
	}

	return nil
}

// outputJSON outputs the analysis result in JSON format for CI integration.
func outputJSON(result *types.AnalysisResult, projectName string, failUnder int) error {
	// Convert issues to JSON format
	jsonIssues := make([]JSONIssue, len(result.Issues))
	summary := IssueSummary{Total: len(result.Issues)}

	for i, issue := range result.Issues {
		jsonIssues[i] = JSONIssue{
			Kind:     string(issue.Kind),
			Severity: issue.Severity.String(),
			File:     issue.File,
			Line:     issue.Line,
			Message:  issue.Message,
		}
		// Count by severity
		switch issue.Severity {
		case types.SeverityCritical:
			summary.Critical++
		case types.SeverityHigh:
			summary.High++
		case types.SeverityMedium:
			summary.Medium++
		case types.SeverityLow:
			summary.Low++
		case types.SeverityInfo:
			summary.Info++
		}
	}

	output := JSONOutput{
		Success: result.HealthScore >= failUnder || failUnder == 0,
		Data: &JSONOutputData{
			HealthScore: result.HealthScore,
			Grade:       types.ScoreGrade(result.HealthScore),
			Issues:      jsonIssues,
			Summary:     summary,
		},
		Meta: &JSONOutputMeta{
			FilesScanned: result.FilesScanned,
			Timestamp:    time.Now().UTC(),
			ProjectName:  projectName,
			FailUnder:    failUnder,
		},
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}

	// Return error if below threshold (for CI exit code)
	if failUnder > 0 && result.HealthScore < failUnder {
		return fmt.Errorf("health score %d is below minimum threshold %d", result.HealthScore, failUnder)
	}

	return nil
}

// outputJSONError outputs an error in JSON format.
func outputJSONError(message string) error {
	output := map[string]interface{}{
		"success": false,
		"error": map[string]string{
			"message": message,
		},
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(output)
	return fmt.Errorf("%s", message)
}

func runCheck(result *types.AnalysisResult, name string, fn func() ([]types.Issue, int, error)) {
	issues, scanned, err := fn()
	result.FilesScanned += scanned
	if err != nil {
		ui.PrintError("Check '" + name + "' failed: " + err.Error())
		return
	}
	result.Issues = append(result.Issues, issues...)
}

func printReport(result *types.AnalysisResult, minSeverity types.Severity) {
	// Group by kind
	kinds := []types.IssueKind{
		types.KindUnhandledError, types.KindMissingValidation,
		types.KindNPlusOne, types.KindHardcodedValue, types.KindDeadRoute,
		types.KindMissingAuth, types.KindCircularImport,
	}

	hasIssues := false
	for _, kind := range kinds {
		kindIssues := result.ByKind(kind)
		if len(kindIssues) == 0 {
			continue
		}
		for _, issue := range kindIssues {
			if issue.Severity > minSeverity {
				continue
			}
			if !hasIssues {
				ui.PrintSection("Issues Found")
				hasIssues = true
			}
			printIssue(issue)
		}
	}

	if !hasIssues {
		fmt.Printf("\n  %s\n", ui.StyleSuccess.Render("No issues found!"))
	}

	// Health score dashboard
	printHealthScore(result)
}

func printIssue(issue types.Issue) {
	var severityStr string
	switch issue.Severity {
	case types.SeverityCritical:
		severityStr = ui.StyleError.Render("[CRITICAL]")
	case types.SeverityHigh:
		severityStr = ui.StyleError.Render("[HIGH]    ")
	case types.SeverityMedium:
		severityStr = ui.StyleWarning.Render("[MEDIUM]  ")
	default:
		severityStr = ui.StyleMuted.Render("[LOW]     ")
	}

	location := issue.File
	if issue.Line > 0 {
		location = fmt.Sprintf("%s:%d", issue.File, issue.Line)
	}
	fmt.Printf("  %s %s\n          %s\n",
		severityStr,
		ui.StyleMuted.Render(location),
		issue.Message,
	)
}

func printHealthScore(result *types.AnalysisResult) {
	score := result.HealthScore
	grade := types.ScoreGrade(score)

	var scoreRendered string
	var gradeRendered string
	switch {
	case score >= 90:
		scoreRendered = ui.StyleSuccess.Render(fmt.Sprintf("%3d/100", score))
		gradeRendered = ui.StyleSuccess.Render(grade)
	case score >= 60:
		scoreRendered = ui.StyleWarning.Render(fmt.Sprintf("%3d/100", score))
		gradeRendered = ui.StyleWarning.Render(grade)
	default:
		scoreRendered = ui.StyleError.Render(fmt.Sprintf("%3d/100", score))
		gradeRendered = ui.StyleError.Render(grade)
	}

	width := 40
	filled := score * width / 100
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	fmt.Printf("\n")
	ui.PrintSection("Health Score")
	fmt.Printf("  %s  %s  Grade: %s\n\n",
		scoreRendered,
		ui.StyleMuted.Render(bar),
		gradeRendered,
	)
	fmt.Printf("  %s scanned  ·  %s issue(s) found\n\n",
		ui.StyleMuted.Render(fmt.Sprintf("%d files", result.FilesScanned)),
		ui.StyleMuted.Render(fmt.Sprintf("%d", len(result.Issues))),
	)

	if score < 60 {
		ui.PrintHint("Run 'go-tk analyze --min-severity=critical' to see only critical issues.")
	}
}

func parseSeverity(s string) types.Severity {
	switch strings.ToLower(s) {
	case "critical":
		return types.SeverityCritical
	case "high":
		return types.SeverityHigh
	case "medium":
		return types.SeverityMedium
	case "info":
		return types.SeverityInfo
	default:
		return types.SeverityLow
	}
}
