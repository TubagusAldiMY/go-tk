package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSeverity_String(t *testing.T) {
	tests := []struct {
		name     string
		severity Severity
		want     string
	}{
		{"critical", SeverityCritical, "CRITICAL"},
		{"high", SeverityHigh, "HIGH"},
		{"medium", SeverityMedium, "MEDIUM"},
		{"low", SeverityLow, "LOW"},
		{"info", SeverityInfo, "INFO"},
		{"unknown defaults to INFO", Severity(99), "INFO"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.severity.String())
		})
	}
}

func TestIssue_String(t *testing.T) {
	tests := []struct {
		name  string
		issue Issue
		want  string
	}{
		{
			name: "with line number",
			issue: Issue{
				Kind:     KindUnhandledError,
				Severity: SeverityHigh,
				File:     "handler.go",
				Line:     42,
				Message:  "error not handled",
			},
			want: "[HIGH] handler.go:42 — error not handled",
		},
		{
			name: "without line number",
			issue: Issue{
				Kind:     KindCircularImport,
				Severity: SeverityCritical,
				File:     "internal/domain",
				Line:     0,
				Message:  "circular import detected",
			},
			want: "[CRITICAL] internal/domain — circular import detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.issue.String())
		})
	}
}

func TestAnalysisResult_ByKind(t *testing.T) {
	result := &AnalysisResult{
		Issues: []Issue{
			{Kind: KindUnhandledError, Message: "err1"},
			{Kind: KindNPlusOne, Message: "n+1"},
			{Kind: KindUnhandledError, Message: "err2"},
			{Kind: KindDeadRoute, Message: "dead"},
		},
	}

	unhandled := result.ByKind(KindUnhandledError)
	assert.Len(t, unhandled, 2)
	assert.Equal(t, "err1", unhandled[0].Message)
	assert.Equal(t, "err2", unhandled[1].Message)

	nplus := result.ByKind(KindNPlusOne)
	assert.Len(t, nplus, 1)

	missing := result.ByKind(KindMissingValidation)
	assert.Len(t, missing, 0)
}

func TestAnalysisResult_BySeverity(t *testing.T) {
	result := &AnalysisResult{
		Issues: []Issue{
			{Severity: SeverityCritical, Message: "critical1"},
			{Severity: SeverityHigh, Message: "high1"},
			{Severity: SeverityMedium, Message: "medium1"},
			{Severity: SeverityLow, Message: "low1"},
			{Severity: SeverityInfo, Message: "info1"},
		},
	}

	// BySeverity returns issues AT OR ABOVE (lower numeric value = higher severity)
	critical := result.BySeverity(SeverityCritical)
	assert.Len(t, critical, 1)

	highAndAbove := result.BySeverity(SeverityHigh)
	assert.Len(t, highAndAbove, 2)

	all := result.BySeverity(SeverityInfo)
	assert.Len(t, all, 5)
}

func TestComputeHealthScore(t *testing.T) {
	tests := []struct {
		name   string
		issues []Issue
		want   int
	}{
		{
			name:   "no issues = perfect score",
			issues: []Issue{},
			want:   100,
		},
		{
			name: "one critical = 85",
			issues: []Issue{
				{Severity: SeverityCritical},
			},
			want: 85,
		},
		{
			name: "one high = 92",
			issues: []Issue{
				{Severity: SeverityHigh},
			},
			want: 92,
		},
		{
			name: "one medium = 96",
			issues: []Issue{
				{Severity: SeverityMedium},
			},
			want: 96,
		},
		{
			name: "one low = 98",
			issues: []Issue{
				{Severity: SeverityLow},
			},
			want: 98,
		},
		{
			name: "info issues = no deduction",
			issues: []Issue{
				{Severity: SeverityInfo},
				{Severity: SeverityInfo},
				{Severity: SeverityInfo},
			},
			want: 100,
		},
		{
			name: "mixed issues",
			issues: []Issue{
				{Severity: SeverityCritical}, // -15
				{Severity: SeverityHigh},     // -8
				{Severity: SeverityMedium},   // -4
				{Severity: SeverityLow},      // -2
			},
			want: 71, // 100 - 15 - 8 - 4 - 2
		},
		{
			name: "floor at zero",
			issues: []Issue{
				{Severity: SeverityCritical},
				{Severity: SeverityCritical},
				{Severity: SeverityCritical},
				{Severity: SeverityCritical},
				{Severity: SeverityCritical},
				{Severity: SeverityCritical},
				{Severity: SeverityCritical}, // 7 critical = -105
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeHealthScore(tt.issues)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestScoreGrade(t *testing.T) {
	tests := []struct {
		score int
		want  string
	}{
		{100, "A"},
		{95, "A"},
		{90, "A"},
		{89, "B"},
		{75, "B"},
		{74, "C"},
		{60, "C"},
		{59, "D"},
		{40, "D"},
		{39, "F"},
		{0, "F"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, ScoreGrade(tt.score))
		})
	}
}
