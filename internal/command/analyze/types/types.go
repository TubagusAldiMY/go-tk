// Package types contains shared types for the analyze command and its checks.
package types

import "fmt"

// Severity levels for analysis issues.
type Severity int

const (
	SeverityCritical Severity = iota
	SeverityHigh
	SeverityMedium
	SeverityLow
	SeverityInfo
)

func (s Severity) String() string {
	switch s {
	case SeverityCritical:
		return "CRITICAL"
	case SeverityHigh:
		return "HIGH"
	case SeverityMedium:
		return "MEDIUM"
	case SeverityLow:
		return "LOW"
	default:
		return "INFO"
	}
}

// IssueKind identifies the type of problem found.
type IssueKind string

const (
	KindDeadRoute         IssueKind = "dead_route"
	KindUnhandledError    IssueKind = "unhandled_error"
	KindMissingValidation IssueKind = "missing_validation"
	KindNPlusOne          IssueKind = "n_plus_one"
	KindHardcodedValue    IssueKind = "hardcoded_value"
	KindMissingAuth       IssueKind = "missing_auth"
	KindCircularImport    IssueKind = "circular_import"
)

// Issue represents a single problem found during analysis.
type Issue struct {
	Kind     IssueKind
	Severity Severity
	File     string
	Line     int
	Message  string
}

func (i Issue) String() string {
	if i.Line > 0 {
		return fmt.Sprintf("[%s] %s:%d — %s", i.Severity, i.File, i.Line, i.Message)
	}
	return fmt.Sprintf("[%s] %s — %s", i.Severity, i.File, i.Message)
}

// AnalysisResult holds all issues found and the computed health score.
type AnalysisResult struct {
	Issues       []Issue
	FilesScanned int
	HealthScore  int // 0–100
}

// ByKind returns all issues of the given kind.
func (r *AnalysisResult) ByKind(kind IssueKind) []Issue {
	var out []Issue
	for _, i := range r.Issues {
		if i.Kind == kind {
			out = append(out, i)
		}
	}
	return out
}

// BySeverity returns issues at or above the given severity.
func (r *AnalysisResult) BySeverity(min Severity) []Issue {
	var out []Issue
	for _, i := range r.Issues {
		if i.Severity <= min {
			out = append(out, i)
		}
	}
	return out
}

// ComputeHealthScore calculates a 0–100 score from a list of issues.
// Deductions: Critical=15, High=8, Medium=4, Low=2, Info=0.
func ComputeHealthScore(issues []Issue) int {
	score := 100
	for _, i := range issues {
		switch i.Severity {
		case SeverityCritical:
			score -= 15
		case SeverityHigh:
			score -= 8
		case SeverityMedium:
			score -= 4
		case SeverityLow:
			score -= 2
		}
	}
	if score < 0 {
		score = 0
	}
	return score
}

// ScoreGrade returns a letter grade for a health score.
func ScoreGrade(score int) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 75:
		return "B"
	case score >= 60:
		return "C"
	case score >= 40:
		return "D"
	default:
		return "F"
	}
}
