package analyze

import "testing"

func TestComputeHealthScore(t *testing.T) {
	tests := []struct {
		name   string
		issues []Issue
		want   int
	}{
		{"no issues", nil, 100},
		{"one critical", []Issue{{Severity: SeverityCritical}}, 85},
		{"one high", []Issue{{Severity: SeverityHigh}}, 92},
		{"mixed", []Issue{
			{Severity: SeverityCritical},
			{Severity: SeverityHigh},
			{Severity: SeverityMedium},
		}, 73},
		{"floor at zero", func() []Issue {
			issues := make([]Issue, 10)
			for i := range issues {
				issues[i] = Issue{Severity: SeverityCritical}
			}
			return issues
		}(), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeHealthScore(tt.issues)
			if got != tt.want {
				t.Errorf("ComputeHealthScore = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestScoreGrade(t *testing.T) {
	cases := []struct {
		score int
		grade string
	}{
		{100, "A"}, {90, "A"}, {89, "B"}, {75, "B"},
		{74, "C"}, {60, "C"}, {59, "D"}, {40, "D"},
		{39, "F"}, {0, "F"},
	}
	for _, c := range cases {
		got := ScoreGrade(c.score)
		if got != c.grade {
			t.Errorf("ScoreGrade(%d) = %q, want %q", c.score, got, c.grade)
		}
	}
}
