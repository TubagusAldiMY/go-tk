// Package test implements the "go-tk test" command.
package test

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/TubagusAldiMY/go-tk/internal/config"
	"github.com/TubagusAldiMY/go-tk/internal/ui"
)

// TestCmd returns the cobra.Command for "go-tk test".
func TestCmd() *cobra.Command {
	var (
		flagMethod       string
		flagRoute        string
		flagOutput       string
		flagBaseURL      string
		flagTimeout      int
		flagGenerateOnly bool
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
  go-tk test --base-url=http://staging:8080`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				flagRoute = args[0]
			}
			return runTest(&testFlags{
				method:       strings.ToUpper(flagMethod),
				route:        flagRoute,
				output:       flagOutput,
				baseURL:      flagBaseURL,
				timeout:      time.Duration(flagTimeout) * time.Second,
				generateOnly: flagGenerateOnly,
			})
		},
	}

	cmd.Flags().StringVar(&flagMethod, "method", "", "Filter by HTTP method (GET, POST, ...)")
	cmd.Flags().StringVar(&flagOutput, "output", "", "Write HTML report to file (e.g. report.html)")
	cmd.Flags().StringVar(&flagBaseURL, "base-url", "", "Base URL of the running server (default: http://localhost:<PORT>)")
	cmd.Flags().IntVar(&flagTimeout, "timeout", 10, "Request timeout in seconds")
	cmd.Flags().BoolVar(&flagGenerateOnly, "generate-only", false, "Print test cases without executing them")

	return cmd
}

type testFlags struct {
	method, route, output, baseURL string
	timeout                        time.Duration
	generateOnly                   bool
}

func runTest(flags *testFlags) error {
	fmt.Println()
	fmt.Println(ui.Banner())
	fmt.Println()

	cwd, _ := os.Getwd()

	// Load project config for PORT.
	cfg, _ := config.Load(cwd) // best-effort

	// Discover routes
	ui.PrintSection("Discovering routes")
	routes, err := DiscoverRoutes(cwd)
	if err != nil {
		return fmt.Errorf("route discovery: %w", err)
	}

	if len(routes) == 0 {
		ui.PrintHint("No routes found. Make sure your router.go is at internal/interfaces/http/router.go")
		return nil
	}

	// Apply filters
	routes = filterRoutes(routes, flags.method, flags.route)
	fmt.Printf("  Found %d route(s)\n", len(routes))

	// Generate test cases
	cases := GenerateTestCases(routes)

	if flags.generateOnly {
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

	ui.PrintSection("Running tests against " + baseURL)

	// Execute tests
	runner := NewRunner(baseURL, flags.timeout)
	results := runner.Run(cases)

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
