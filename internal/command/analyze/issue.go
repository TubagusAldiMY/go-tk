// Package analyze implements the "go-tk analyze" command.
package analyze

import "github.com/TubagusAldiMY/go-tk/internal/command/analyze/types"

// Re-export types from the types sub-package so callers can use the analyze package directly.

// Severity levels for analysis issues.
type Severity = types.Severity

const (
	SeverityCritical = types.SeverityCritical
	SeverityHigh     = types.SeverityHigh
	SeverityMedium   = types.SeverityMedium
	SeverityLow      = types.SeverityLow
	SeverityInfo     = types.SeverityInfo
)

// IssueKind identifies the type of problem found.
type IssueKind = types.IssueKind

const (
	KindDeadRoute         = types.KindDeadRoute
	KindUnhandledError    = types.KindUnhandledError
	KindMissingValidation = types.KindMissingValidation
	KindNPlusOne          = types.KindNPlusOne
	KindHardcodedValue    = types.KindHardcodedValue
	KindMissingAuth       = types.KindMissingAuth
	KindCircularImport    = types.KindCircularImport
)

// Issue represents a single problem found during analysis.
type Issue = types.Issue

// AnalysisResult holds all issues found and the computed health score.
type AnalysisResult = types.AnalysisResult
