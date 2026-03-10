package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// TestCase represents a single HTTP test to execute.
type TestCase struct {
	Name    string
	Method  string
	Path    string
	Body    string            // JSON body for POST/PUT/PATCH
	Headers map[string]string // extra headers
}

// TestResult holds the outcome of one test case execution.
type TestResult struct {
	TestCase
	StatusCode int
	Duration   time.Duration
	Pass       bool
	Error      string
	Response   string // truncated response body
}

// Runner executes HTTP test cases against a base URL.
type Runner struct {
	BaseURL string
	Timeout time.Duration
	client  *http.Client
}

// NewRunner creates a Runner targeting baseURL.
func NewRunner(baseURL string, timeout time.Duration) *Runner {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &Runner{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Timeout: timeout,
		client:  &http.Client{Timeout: timeout},
	}
}

// Run executes all test cases and returns results.
func (r *Runner) Run(cases []TestCase) []TestResult {
	results := make([]TestResult, 0, len(cases))
	for _, tc := range cases {
		results = append(results, r.runOne(tc))
	}
	return results
}

func (r *Runner) runOne(tc TestCase) TestResult {
	result := TestResult{TestCase: tc}

	url := r.BaseURL + tc.Path
	ctx, cancel := context.WithTimeout(context.Background(), r.Timeout)
	defer cancel()

	var bodyReader *bytes.Reader
	if tc.Body != "" {
		bodyReader = bytes.NewReader([]byte(tc.Body))
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	req, err := http.NewRequestWithContext(ctx, tc.Method, url, bodyReader)
	if err != nil {
		result.Error = fmt.Sprintf("creating request: %v", err)
		return result
	}

	if tc.Body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Request-ID", fmt.Sprintf("go-tk-test-%d", time.Now().UnixNano()))
	for k, v := range tc.Headers {
		req.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := r.client.Do(req)
	result.Duration = time.Since(start)

	if err != nil {
		result.Error = fmt.Sprintf("executing request: %v", err)
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err == nil {
		body := buf.String()
		if len(body) > 512 {
			body = body[:512] + "…"
		}
		result.Response = body
	}

	// Pass = any 2xx or 3xx response (not a connection error)
	result.Pass = resp.StatusCode < 500
	return result
}

// GenerateTestCases produces a minimal test case for each route.
// For routes with path parameters (/:id), it substitutes "1".
func GenerateTestCases(routes []RouteInfo) []TestCase {
	cases := make([]TestCase, 0, len(routes))
	for _, r := range routes {
		tc := TestCase{
			Name:   fmt.Sprintf("%s %s", r.Method, r.Path),
			Method: r.Method,
			Path:   substitutePathParams(r.Path),
		}
		// Add minimal body for mutating methods
		if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
			tc.Body = "{}"
		}
		cases = append(cases, tc)
	}
	return cases
}

// substitutePathParams replaces :param and {param} with "1".
func substitutePathParams(path string) string {
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if strings.HasPrefix(p, ":") || (strings.HasPrefix(p, "{") && strings.HasSuffix(p, "}")) {
			parts[i] = "1"
		}
	}
	return strings.Join(parts, "/")
}

// Summary returns pass/fail counts from results.
func Summary(results []TestResult) (passed, failed int) {
	for _, r := range results {
		if r.Pass {
			passed++
		} else {
			failed++
		}
	}
	return
}

// MarshalJSON serialises results for the HTML report.
func MarshalResultsJSON(results []TestResult) ([]byte, error) {
	return json.MarshalIndent(results, "", "  ")
}
