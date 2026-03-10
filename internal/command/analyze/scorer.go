package analyze

import "github.com/TubagusAldiMY/go-tk/internal/command/analyze/types"

// ComputeHealthScore delegates to the types package implementation.
func ComputeHealthScore(issues []Issue) int {
	return types.ComputeHealthScore(issues)
}

// ScoreGrade returns a letter grade for a health score.
func ScoreGrade(score int) string {
	return types.ScoreGrade(score)
}
