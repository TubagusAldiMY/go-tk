// Package env implements the "go-tk env" command family.
package env

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/TubagusAldiMY/go-tk/internal/config"
	"github.com/TubagusAldiMY/go-tk/internal/ui"
)

const (
	defaultEnvFile     = ".env"
	defaultExampleFile = ".env.example"
	fmtSingleItem      = "  %s\n"
	fmtTableRow        = "  %s %s   %s\n"
)

// EnvCmd returns the cobra.Command for "go-tk env".
func EnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Environment variable management (validate, sync, generate-example, check)",
		Long: `Manage .env files for your go-tk project.

Validates your .env against .env.example, syncs missing keys,
and provides pre-deploy preflight checks including optional DB connectivity (--db).`,
	}

	cmd.AddCommand(validateCmd())
	cmd.AddCommand(syncCmd())
	cmd.AddCommand(generateExampleCmd())
	cmd.AddCommand(checkCmd())
	cmd.AddCommand(dbCheckCmd())

	return cmd
}

// --- validate ---

func validateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate .env against .env.example",
		Long:  `Compare .env with .env.example and report missing, empty, or extra variables.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd := mustCwd()
			envFile := filepath.Join(cwd, defaultEnvFile)
			exampleFile := filepath.Join(cwd, defaultExampleFile)

			if err := requireFile(exampleFile); err != nil {
				return err
			}

			ui.PrintSection("Validating environment variables")

			result, err := Validate(envFile, exampleFile)
			if err != nil {
				return err
			}

			if !ui.Quiet {
				printValidationResult(result)
			}

			if !result.IsOK() {
				return fmt.Errorf("%d missing, %d empty required variable(s)", len(result.Missing), len(result.Empty))
			}
			return nil
		},
	}
}

// --- sync ---

func syncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Add missing keys from .env.example into .env (with empty values)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd := mustCwd()
			envFile := filepath.Join(cwd, defaultEnvFile)
			exampleFile := filepath.Join(cwd, defaultExampleFile)

			if err := requireFile(exampleFile); err != nil {
				return err
			}

			ui.PrintSection("Syncing .env from .env.example")

			added, err := Sync(envFile, exampleFile)
			if err != nil {
				return err
			}

			if len(added) == 0 {
				ui.PrintHint(".env is already in sync with .env.example.")
				return nil
			}

			if !ui.Quiet {
				for _, k := range added {
					fmt.Printf("  %s %s=\n", ui.StyleSuccess.Render("+"), k)
				}
			}
			ui.PrintDone(fmt.Sprintf("%d key(s) added to .env.", len(added)))
			ui.PrintHint("Fill in the values before running the application.")
			return nil
		},
	}
}

// --- generate-example ---

func generateExampleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "generate-example",
		Short: "Generate .env.example from .env (strips values)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd := mustCwd()
			envFile := filepath.Join(cwd, defaultEnvFile)
			exampleFile := filepath.Join(cwd, defaultExampleFile)

			if err := requireFile(envFile); err != nil {
				return fmt.Errorf(".env not found — create it first")
			}

			ui.PrintSection("Generating .env.example")

			if err := GenerateExample(envFile, exampleFile); err != nil {
				return err
			}

			ui.PrintFileCreated(exampleFile)
			ui.PrintDone(".env.example updated.")
			ui.PrintHint("Commit .env.example — never commit .env itself.")
			return nil
		},
	}
}

// --- check (pre-deploy preflight) ---

func checkCmd() *cobra.Command {
	var includeDB bool

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Pre-deploy preflight check (strict — exits non-zero on any issue)",
		Long: `Run a strict pre-deploy check:
- All required variables must be present and non-empty
- No variables may have placeholder values (e.g. "change-me")
- Optionally test database TCP connectivity with --db

Exits with code 1 if any check fails, making it safe to use in CI/CD pipelines.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd := mustCwd()
			envFile := filepath.Join(cwd, defaultEnvFile)
			exampleFile := filepath.Join(cwd, defaultExampleFile)

			if err := requireFile(exampleFile); err != nil {
				return err
			}

			ui.PrintSection("Pre-deploy preflight check")

			result, err := Validate(envFile, exampleFile)
			if err != nil {
				return err
			}

			placeholders := collectPlaceholders(result.Vars)

			if !ui.Quiet {
				printValidationResult(result)
				printPlaceholderWarnings(placeholders)
			}

			if !result.IsOK() || len(placeholders) > 0 {
				fmt.Println()
				ui.PrintError("Preflight check FAILED — deployment blocked.")
				return fmt.Errorf("environment not ready for deployment")
			}

			if includeDB {
				if err := runDBCheck(5); err != nil {
					return fmt.Errorf("database check failed: %w", err)
				}
			}

			fmt.Println()
			ui.PrintDone("Preflight check PASSED — environment is ready.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&includeDB, "db", false, "Also test database TCP connectivity (replaces env db-check)")
	return cmd
}

// collectPlaceholders returns keys whose raw values match known placeholder patterns.
func collectPlaceholders(vars []VarResult) []string {
	var placeholders []string
	for _, v := range vars {
		if isPlaceholder(v.RawValue) {
			placeholders = append(placeholders, v.Key)
		}
	}
	return placeholders
}

// printPlaceholderWarnings prints a warning line for each placeholder key.
func printPlaceholderWarnings(placeholders []string) {
	if len(placeholders) == 0 {
		return
	}
	fmt.Println()
	for _, k := range placeholders {
		fmt.Printf("  %s %s contains placeholder value\n", ui.StyleWarning.Render("!"), k)
	}
}

// --- helpers ---

func printValidationResult(result *ValidationResult) {
	maxLen := 0
	for _, v := range result.Vars {
		if len(v.Key) > maxLen {
			maxLen = len(v.Key)
		}
	}
	fmt.Println(repeatStr("─", 50))

	for _, v := range result.Vars {
		pad := repeatStr(" ", maxLen-len(v.Key))
		formatVarRow(v, v.Key+pad)
	}

	if len(result.Extra) > 0 {
		fmt.Println()
		for _, k := range result.Extra {
			fmt.Printf(fmtTableRow, ui.StyleMuted.Render("?"), k, ui.StyleMuted.Render("(not in .env.example)"))
		}
	}

	fmt.Println(repeatStr("─", 50))
	printValidationSummary(result)
}

// formatVarRow prints a single variable row based on its status.
func formatVarRow(v VarResult, keyWithPad string) {
	switch v.Status {
	case VarPresent:
		fmt.Printf("  %s %s = %s\n", ui.StyleSuccess.Render("✓"), keyWithPad, ui.StyleMuted.Render(v.Value))
	case VarOptional:
		fmt.Printf(fmtTableRow, ui.StyleMuted.Render("~"), keyWithPad, ui.StyleMuted.Render("(optional, not set)"))
	case VarMissing:
		fmt.Printf(fmtTableRow, ui.StyleError.Render("✗"), keyWithPad, ui.StyleError.Render("MISSING"))
	case VarEmpty:
		fmt.Printf(fmtTableRow, ui.StyleWarning.Render("!"), keyWithPad, ui.StyleWarning.Render("empty"))
	}
}

// printValidationSummary prints the summary line after the variable table.
func printValidationSummary(result *ValidationResult) {
	total := len(result.Vars)
	missing := len(result.Missing)
	empty := len(result.Empty)
	issues := missing + empty
	if issues > 0 {
		msg := fmt.Sprintf("%d/%d required variables have issues", issues, total)
		if missing > 0 {
			msg += fmt.Sprintf(" (%d missing", missing)
			if empty > 0 {
				msg += fmt.Sprintf(", %d empty", empty)
			}
			msg += ")"
		} else {
			msg += fmt.Sprintf(" (%d empty)", empty)
		}
		fmt.Printf(fmtSingleItem, ui.StyleError.Render(msg))
		fmt.Printf(fmtSingleItem, ui.StyleMuted.Render("Run: go-tk env sync"))
	} else {
		fmt.Printf(fmtSingleItem, ui.StyleSuccess.Render(fmt.Sprintf("All %d required variables set", total)))
	}
}

func requireFile(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", path)
	}
	return nil
}

func mustCwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return cwd
}

func isPlaceholder(val string) bool {
	lower := fmt.Sprintf("%s", val)
	placeholders := []string{"change-me", "changeme", "todo", "replace-me", "your-secret", "xxxxx", "example"}
	for _, p := range placeholders {
		if len(lower) > 0 && contains(lower, p) {
			return true
		}
	}
	return false
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}

func repeatStr(s string, n int) string {
	if n <= 0 {
		return ""
	}
	result := make([]byte, len(s)*n)
	for i := 0; i < n; i++ {
		copy(result[i*len(s):], s)
	}
	return string(result)
}

// loadConfig is a convenience to load project config (used by commands that need it).
func loadConfig() (*config.Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}
	return config.Load(cwd)
}

// --- db-check (database connectivity check) ---

func dbCheckCmd() *cobra.Command {
	var timeout int

	cmd := &cobra.Command{
		Use:        "db-check",
		Short:      "Test database connectivity using .env credentials",
		Deprecated: "use 'go-tk env check --db' instead",
		Long: `Pre-flight database connectivity check.

Tests that the database is reachable using credentials from .env.
Useful before running migrations or deployment.

Environment variables read:
  DB_DRIVER   - postgres | mysql
  DB_HOST     - Database host
  DB_PORT     - Database port
  DB_USER     - Database user
  DB_PASSWORD - Database password
  DB_NAME     - Database name
  DB_SSL_MODE - SSL mode (for postgres)

Example:
  go-tk env db-check
  go-tk env db-check --timeout=10`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDBCheck(timeout)
		},
	}

	cmd.Flags().IntVar(&timeout, "timeout", 5, "Connection timeout in seconds")

	return cmd
}

func runDBCheck(timeoutSec int) error {
	cwd := mustCwd()
	envFile := filepath.Join(cwd, defaultEnvFile)

	if err := requireFile(envFile); err != nil {
		return fmt.Errorf(".env not found — create it first")
	}

	ui.PrintSection("Database connectivity check")

	envVars, err := parseEnvFile(envFile)
	if err != nil {
		return fmt.Errorf("parsing .env: %w", err)
	}

	c, missing := extractDBVars(envVars)
	if len(missing) > 0 {
		return reportMissingDBVars(missing)
	}
	c.timeoutSec = timeoutSec

	if !ui.Quiet {
		printDBInfo(c)
		fmt.Printf("  Testing connection...")
	}

	if _, err := buildDSN(c); err != nil {
		return err
	}

	if err := testTCPConnection(c.host, c.port, timeoutSec); err != nil {
		if !ui.Quiet {
			fmt.Printf(" %s\n\n  %s\n", ui.StyleError.Render("FAILED"), ui.StyleError.Render("Database is unreachable: "+err.Error()))
			ui.PrintHint("Check that the database is running and network is accessible.")
			ui.PrintHint("Verify DB_HOST and DB_PORT in .env")
		}
		return fmt.Errorf("database connectivity check failed: %s", err.Error())
	}

	if !ui.Quiet {
		fmt.Printf(" %s\n\n", ui.StyleSuccess.Render("OK"))
	}
	ui.PrintDone("Database is reachable at " + c.host + ":" + c.port)
	ui.PrintHint("Note: This only tests TCP connectivity. Run 'go-tk migrate status' to verify credentials.")
	return nil
}

// dbConnConfig holds the resolved database connection parameters.
type dbConnConfig struct {
	driver, host, port, user, password, dbname, sslmode string
	timeoutSec                                          int
}

// extractDBVars pulls required DB config values from the env map and lists any that are missing.
func extractDBVars(envVars map[string]string) (dbConnConfig, []string) {
	c := dbConnConfig{
		driver:   envVars["DB_DRIVER"],
		host:     envVars["DB_HOST"],
		port:     envVars["DB_PORT"],
		user:     envVars["DB_USER"],
		password: envVars["DB_PASSWORD"],
		dbname:   envVars["DB_NAME"],
		sslmode:  envVars["DB_SSL_MODE"],
	}

	var missing []string
	for _, check := range []struct{ val, name string }{
		{c.driver, "DB_DRIVER"}, {c.host, "DB_HOST"}, {c.port, "DB_PORT"},
		{c.user, "DB_USER"}, {c.dbname, "DB_NAME"},
	} {
		if check.val == "" {
			missing = append(missing, check.name)
		}
	}
	return c, missing
}

// reportMissingDBVars prints and returns an error for missing DB variables.
func reportMissingDBVars(missing []string) error {
	if !ui.Quiet {
		for _, m := range missing {
			fmt.Printf("  %s %s not set in .env\n", ui.StyleError.Render("✗"), m)
		}
	}
	return fmt.Errorf("missing required database environment variables: %s", strings.Join(missing, ", "))
}

// printDBInfo prints the resolved DB connection parameters.
func printDBInfo(c dbConnConfig) {
	fmt.Printf("  Driver:   %s\n", c.driver)
	fmt.Printf("  Host:     %s:%s\n", c.host, c.port)
	fmt.Printf("  Database: %s\n", c.dbname)
	fmt.Printf("  User:     %s\n", c.user)
	fmt.Printf("  Timeout:  %ds\n", c.timeoutSec)
	fmt.Println()
}

// buildDSN constructs the driver-specific connection string.
func buildDSN(c dbConnConfig) (string, error) {
	switch c.driver {
	case "postgres":
		sslmode := c.sslmode
		if sslmode == "" {
			sslmode = "disable"
		}
		return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d",
			c.host, c.port, c.user, c.password, c.dbname, sslmode, c.timeoutSec), nil
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?timeout=%ds&parseTime=true",
			c.user, c.password, c.host, c.port, c.dbname, c.timeoutSec), nil
	default:
		return "", fmt.Errorf("unsupported DB_DRIVER: %s (valid: postgres, mysql)", c.driver)
	}
}

// testTCPConnection tests if a TCP connection can be established to host:port.
func testTCPConnection(host, port string, timeoutSec int) error {
	addr := fmt.Sprintf("%s:%s", host, port)
	conn, err := net.DialTimeout("tcp", addr, time.Duration(timeoutSec)*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	return nil
}
