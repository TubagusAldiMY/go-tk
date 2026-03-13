// Package test implements the "go-tk test" command.
package test

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/TubagusAldiMY/go-tk/internal/config"
	"github.com/TubagusAldiMY/go-tk/internal/ui"
)

// JSONTestOutput is the machine-readable output format for CI integration.
type JSONTestOutput struct {
	Success bool             `json:"success"`
	Data    *JSONTestData    `json:"data"`
	Meta    *JSONTestMeta    `json:"meta"`
}

// JSONTestData contains the test results.
type JSONTestData struct {
	TotalTests  int              `json:"total_tests"`
	Passed      int              `json:"passed"`
	Failed      int              `json:"failed"`
	PassRate    float64          `json:"pass_rate"`
	Results     []JSONTestResult `json:"results"`
}

// JSONTestResult represents a single test result.
type JSONTestResult struct {
	Method       string  `json:"method"`
	Path         string  `json:"path"`
	StatusCode   int     `json:"status_code"`
	Expected     int     `json:"expected"`
	Passed       bool    `json:"passed"`
	DurationMs   float64 `json:"duration_ms"`
	Error        string  `json:"error,omitempty"`
}

// JSONTestMeta contains metadata about the test run.
type JSONTestMeta struct {
	BaseURL       string    `json:"base_url"`
	Timestamp     time.Time `json:"timestamp"`
	TimeoutSec    int       `json:"timeout_sec"`
	RoutesFound   int       `json:"routes_found"`
}

// TestCmd returns the cobra.Command for "go-tk test".
func TestCmd() *cobra.Command {
	var (
		flagMethod       string
		flagRoute        string
		flagOutput       string
		flagBaseURL      string
		flagTimeout      int
		flagGenerateOnly bool
		flagFormat       string
	)

	cmd := &cobra.Command{
		Use:   "test [route]",
		Short: "Auto-generate and run HTTP tests from route definitions",
		Long: `Discover routes from your router.go using Go AST and execute HTTP tests.

Examples:
  go-tk test                          # Test all routes
  go-tk test /api/v1/products         # Test one specific route
  go-tk test --method=GET             # Only GET routes
  go-tk test --generate-only          # Print test cases without executing
  go-tk test --output=report.html     # Save HTML report
  go-tk test --format=json            # Output JSON for CI integration
  go-tk test --base-url=http://staging:8080`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				flagRoute = args[0]
			}
			return runTest(&testFlags{
				method:       strings.ToUpper(flagMethod),
				route:        flagRoute,
				output:       flagOutput,
				format:       flagFormat,
				baseURL:      flagBaseURL,
				timeout:      time.Duration(flagTimeout) * time.Second,
				generateOnly: flagGenerateOnly,
			})
		},
	}

	cmd.Flags().StringVar(&flagMethod, "method", "", "Filter by HTTP method (GET, POST, ...)")
	cmd.Flags().StringVar(&flagOutput, "output", "", "Write HTML report to file (e.g. report.html)")
	cmd.Flags().StringVarP(&flagFormat, "format", "f", "text", "Output format: text | json")
	cmd.Flags().StringVar(&flagBaseURL, "base-url", "", "Base URL of the running server (default: http://localhost:<PORT>)")
	cmd.Flags().IntVar(&flagTimeout, "timeout", 10, "Request timeout in seconds")
	cmd.Flags().BoolVar(&flagGenerateOnly, "generate-only", false, "Print test cases without executing them")

	return cmd
}

type testFlags struct {
	method, route, output, format, baseURL string
	timeout                                time.Duration
	generateOnly                           bool
}

func runTest(flags *testFlags) error {
	// Validate output format
	if flags.format != "text" && flags.format != "json" {
		return fmt.Errorf("invalid format: %s (valid: text, json)", flags.format)
	}

	cwd, _ := os.Getwd()

	// Load project config for PORT.
	cfg, _ := config.Load(cwd) // best-effort

	// Skip banner for JSON output
	if flags.format != "json" {
		fmt.Println()
		fmt.Println(ui.Banner())
		fmt.Println()
	}

	// Discover routes
	if flags.format != "json" {
		ui.PrintSection("Discovering routes")
	}
	routes, err := DiscoverRoutes(cwd)
	if err != nil {
		if flags.format == "json" {
			return outputTestJSONError("route discovery: " + err.Error())
		}
		return fmt.Errorf("route discovery: %w", err)
	}

	if len(routes) == 0 {
		if flags.format == "json" {
			return outputTestJSONError("no routes found")
		}
		ui.PrintHint("No routes found. Make sure your router.go is at internal/interfaces/http/router.go")
		return nil
	}

	// Apply filters
	routes = filterRoutes(routes, flags.method, flags.route)
	if flags.format != "json" {
		fmt.Printf("  Found %d route(s)\n", len(routes))
	}

	// Generate test cases
	cases := GenerateTestCases(routes)

	if flags.generateOnly {
		if flags.format == "json" {
			// Output generated test cases as JSON
			output := map[string]interface{}{
				"success": true,
				"data": map[string]interface{}{
					"test_cases": cases,
					"count":      len(cases),
				},
			}
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}
		ui.PrintSection("Generated test cases")
		for _, tc := range cases {
			body := ""
			if tc.Body != "" {
				body = "  body: " + tc.Body
			}
			fmt.Printf("  %s %s%s\n", ui.StyleMuted.Render(fmt.Sprintf("%-8s", tc.Method)), tc.Path, body)
		}
		return nil
	}

	// Resolve base URL
	baseURL := flags.baseURL
	if baseURL == "" {
		port := "8080"
		if cfg != nil {
			// Try to read PORT from .env
			if envPort := readEnvPort(cwd); envPort != "" {
				port = envPort
			}
		}
		baseURL = "http://localhost:" + port
	}

	if flags.format != "json" {
		ui.PrintSection("Running tests against " + baseURL)
	}

	// Execute tests
	runner := NewRunner(baseURL, flags.timeout)
	results := runner.Run(cases)

	// Output based on format
	if flags.format == "json" {
		return outputTestJSON(results, baseURL, len(routes), int(flags.timeout.Seconds()))
	}

	// Print terminal report
	PrintTerminalReport(results)

	// Write HTML report if requested
	if flags.output != "" {
		if err := WriteHTMLReport(results, flags.output); err != nil {
			ui.PrintError("Writing HTML report: " + err.Error())
		} else {
			ui.PrintHint("HTML report written to: " + flags.output)
		}
	}

	// Exit with error if any tests failed
	_, failed := Summary(results)
	if failed > 0 {
		return fmt.Errorf("%d test(s) failed", failed)
	}
	return nil
}

// outputTestJSON outputs the test results in JSON format.
func outputTestJSON(results []TestResult, baseURL string, routesFound, timeoutSec int) error {
	passed, failed := Summary(results)
	total := passed + failed
	passRate := 0.0
	if total > 0 {
		passRate = float64(passed) / float64(total) * 100
	}

	jsonResults := make([]JSONTestResult, len(results))
	for i, r := range results {
		jsonResults[i] = JSONTestResult{
			Method:     r.Method,
			Path:       r.Path,
			StatusCode: r.StatusCode,
			Expected:   200, // Default expected status
			Passed:     r.Pass,
			DurationMs: float64(r.Duration.Milliseconds()),
			Error:      r.Error,
		}
	}

	output := JSONTestOutput{
		Success: failed == 0,
		Data: &JSONTestData{
			TotalTests: total,
			Passed:     passed,
			Failed:     failed,
			PassRate:   passRate,
			Results:    jsonResults,
		},
		Meta: &JSONTestMeta{
			BaseURL:     baseURL,
			Timestamp:   time.Now().UTC(),
			TimeoutSec:  timeoutSec,
			RoutesFound: routesFound,
		},
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}

	if failed > 0 {
		return fmt.Errorf("%d test(s) failed", failed)
	}
	return nil
}

// outputTestJSONError outputs an error in JSON format.
func outputTestJSONError(message string) error {
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

func filterRoutes(routes []RouteInfo, method, path string) []RouteInfo {
	if method == "" && path == "" {
		return routes
	}
	filtered := routes[:0]
	for _, r := range routes {
		if method != "" && r.Method != method {
			continue
		}
		if path != "" && r.Path != path {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

// readEnvPort reads PORT from .env file in projectRoot.
func readEnvPort(projectRoot string) string {
	data, err := os.ReadFile(projectRoot + "/.env")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "PORT=") {
			return strings.TrimPrefix(line, "PORT=")
		}
	}
	return ""
}
