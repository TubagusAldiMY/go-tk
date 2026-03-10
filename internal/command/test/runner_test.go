package test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRunnerSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer srv.Close()

	runner := NewRunner(srv.URL, 5*time.Second)
	results := runner.Run([]TestCase{
		{Name: "GET /health", Method: "GET", Path: "/health"},
	})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Pass {
		t.Errorf("expected pass, got fail: %v", results[0].Error)
	}
	if results[0].StatusCode != 200 {
		t.Errorf("expected 200, got %d", results[0].StatusCode)
	}
}

func TestRunnerServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	runner := NewRunner(srv.URL, 5*time.Second)
	results := runner.Run([]TestCase{
		{Name: "GET /broken", Method: "GET", Path: "/broken"},
	})

	if results[0].Pass {
		t.Error("expected fail for 5xx response")
	}
}

func TestRunnerConnectionRefused(t *testing.T) {
	runner := NewRunner("http://localhost:19999", 1*time.Second)
	results := runner.Run([]TestCase{
		{Name: "GET /", Method: "GET", Path: "/"},
	})

	if results[0].Pass {
		t.Error("expected fail for connection refused")
	}
	if results[0].Error == "" {
		t.Error("expected error message for connection refused")
	}
}

func TestSubstitutePathParams(t *testing.T) {
	cases := []struct{ input, want string }{
		{"/api/v1/users/:id", "/api/v1/users/1"},
		{"/items/{id}/reviews/{reviewId}", "/items/1/reviews/1"},
		{"/health", "/health"},
		{"/api/v1/products", "/api/v1/products"},
	}
	for _, c := range cases {
		got := substitutePathParams(c.input)
		if got != c.want {
			t.Errorf("substitutePathParams(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestGenerateTestCases(t *testing.T) {
	routes := []RouteInfo{
		{Method: "GET", Path: "/api/v1/users"},
		{Method: "POST", Path: "/api/v1/users"},
		{Method: "GET", Path: "/api/v1/users/:id"},
		{Method: "DELETE", Path: "/api/v1/users/:id"},
	}
	cases := GenerateTestCases(routes)

	if len(cases) != 4 {
		t.Fatalf("expected 4 cases, got %d", len(cases))
	}

	// POST should have a body
	var postCase *TestCase
	for i := range cases {
		if cases[i].Method == "POST" {
			postCase = &cases[i]
		}
	}
	if postCase == nil {
		t.Fatal("POST case not found")
	}
	if postCase.Body == "" {
		t.Error("POST case should have a body")
	}

	// Path params should be substituted
	for _, tc := range cases {
		if tc.Path != substitutePathParams(tc.Path) {
			t.Errorf("path params not substituted in %q", tc.Path)
		}
	}
}

func TestSummary(t *testing.T) {
	results := []TestResult{
		{Pass: true},
		{Pass: true},
		{Pass: false},
	}
	passed, failed := Summary(results)
	if passed != 2 || failed != 1 {
		t.Errorf("Summary = (%d, %d), want (2, 1)", passed, failed)
	}
}
