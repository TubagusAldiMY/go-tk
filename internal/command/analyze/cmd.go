// Package analyze implements the "go-tk analyze" command.
package analyze

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TubagusAldiMY/go-tk/internal/command/analyze/checks"
	"github.com/TubagusAldiMY/go-tk/internal/command/analyze/types"
	"github.com/TubagusAldiMY/go-tk/internal/config"
	"github.com/TubagusAldiMY/go-tk/internal/ui"
)

// AnalyzeCmd returns the cobra.Command for "go-tk analyze".
func AnalyzeCmd() *cobra.Command {
	var minSeverity string
	var failUnder int

	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze project code health (dead routes, N+1, missing validation, ...)",
		Long: `Static analysis of your Go backend project.

Checks performed:
  • Unhandled errors (errors discarded with _)
  • Missing input validation (bind without validate)
  • N+1 query patterns (DB call inside loop)
  • Hardcoded credentials and config values

Outputs a health score (0–100) and a graded report.

Use --fail-under in CI pipelines to enforce a minimum health score:
  go-tk analyze --fail-under=75`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnalyze(minSeverity, failUnder)
		},
	}

	cmd.Flags().StringVar(&minSeverity, "min-severity", "low", "Minimum severity to report (critical|high|medium|low|info)")
	cmd.Flags().IntVar(&failUnder, "fail-under", 0, "Exit with code 1 if health score is below this threshold (0 = disabled)")

	return cmd
}

func runAnalyze(minSeverityStr string, failUnder int) error {
	fmt.Println()
	fmt.Println(ui.Banner())
	fmt.Println()

	cwd, _ := os.Getwd()
	cfg, err := config.Load(cwd)
	if err != nil {
		return config.ErrConfigNotFound
	}

	ui.PrintSection("Analyzing project: " + cfg.Project.Name)

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

	// Compute health score
	result.HealthScore = types.ComputeHealthScore(result.Issues)

	// Print report
	printReport(result, parseSeverity(minSeverityStr))

	// Enforce minimum health score for CI pipelines.
	if failUnder > 0 && result.HealthScore < failUnder {
		return fmt.Errorf("health score %d is below minimum threshold %d", result.HealthScore, failUnder)
	}

	return nil
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
