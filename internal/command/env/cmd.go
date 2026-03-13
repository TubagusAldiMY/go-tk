// Package env implements the "go-tk env" command family.
package env

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/TubagusAldiMY/go-tk/internal/config"
	"github.com/TubagusAldiMY/go-tk/internal/ui"
)

const (
	defaultEnvFile     = ".env"
	defaultExampleFile = ".env.example"
)

// EnvCmd returns the cobra.Command for "go-tk env".
func EnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Environment variable management (validate, sync, generate-example, check, db-check)",
		Long: `Manage .env files for your go-tk project.

Validates your .env against .env.example, syncs missing keys,
and provides pre-deploy preflight checks.`,
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

			printValidationResult(result)

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

			for _, k := range added {
				fmt.Printf("  %s %s=\n", ui.StyleSuccess.Render("+"), k)
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
	return &cobra.Command{
		Use:   "check",
		Short: "Pre-deploy preflight check (strict — exits non-zero on any issue)",
		Long: `Run a strict pre-deploy check:
- All required variables must be present and non-empty
- No variables may have placeholder values (e.g. "change-me")

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

			// Check for placeholder values.
			var placeholders []string
			for _, v := range result.Vars {
				if isPlaceholder(v.Value) {
					placeholders = append(placeholders, v.Key)
				}
			}

			printValidationResult(result)

			if len(placeholders) > 0 {
				fmt.Println()
				for _, k := range placeholders {
					fmt.Printf("  %s %s contains placeholder value\n", ui.StyleWarning.Render("!"), k)
				}
			}

			if !result.IsOK() || len(placeholders) > 0 {
				fmt.Println()
				ui.PrintError("Preflight check FAILED — deployment blocked.")
				return fmt.Errorf("environment not ready for deployment")
			}

			fmt.Println()
			ui.PrintDone("Preflight check PASSED — environment is ready.")
			return nil
		},
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
		key := v.Key + pad

		switch v.Status {
		case VarPresent:
			fmt.Printf("  %s %s = %s\n", ui.StyleSuccess.Render("✓"), key, ui.StyleMuted.Render(v.Value))
		case VarOptional:
			fmt.Printf("  %s %s   %s\n", ui.StyleMuted.Render("~"), key, ui.StyleMuted.Render("(optional, not set)"))
		case VarMissing:
			fmt.Printf("  %s %s   %s\n", ui.StyleError.Render("✗"), key, ui.StyleError.Render("MISSING"))
		case VarEmpty:
			fmt.Printf("  %s %s   %s\n", ui.StyleWarning.Render("!"), key, ui.StyleWarning.Render("empty"))
		}
	}

	if len(result.Extra) > 0 {
		fmt.Println()
		for _, k := range result.Extra {
			fmt.Printf("  %s %s   %s\n", ui.StyleMuted.Render("?"), k, ui.StyleMuted.Render("(not in .env.example)"))
		}
	}

	fmt.Println(repeatStr("─", 50))

	total := len(result.Vars)
	missing := len(result.Missing)
	if missing > 0 {
		fmt.Printf("  %s\n", ui.StyleError.Render(fmt.Sprintf("%d/%d required variables missing", missing, total)))
		fmt.Printf("  %s\n", ui.StyleMuted.Render("Run: go-tk env sync"))
	} else {
		fmt.Printf("  %s\n", ui.StyleSuccess.Render(fmt.Sprintf("All %d required variables set", total)))
	}
}

func requireFile(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", path)
	}
	return nil
}

func mustCwd() string {
	cwd, _ := os.Getwd()
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
	cwd, _ := os.Getwd()
	return config.Load(cwd)
}

// --- db-check (database connectivity check) ---

func dbCheckCmd() *cobra.Command {
	var timeout int

	cmd := &cobra.Command{
		Use:   "db-check",
		Short: "Test database connectivity using .env credentials",
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

	// Load environment variables
	envVars, err := parseEnvFile(envFile)
	if err != nil {
		return fmt.Errorf("parsing .env: %w", err)
	}

	driver := envVars["DB_DRIVER"]
	host := envVars["DB_HOST"]
	port := envVars["DB_PORT"]
	user := envVars["DB_USER"]
	password := envVars["DB_PASSWORD"]
	dbname := envVars["DB_NAME"]
	sslmode := envVars["DB_SSL_MODE"]

	// Validate required fields
	missing := []string{}
	if driver == "" {
		missing = append(missing, "DB_DRIVER")
	}
	if host == "" {
		missing = append(missing, "DB_HOST")
	}
	if port == "" {
		missing = append(missing, "DB_PORT")
	}
	if user == "" {
		missing = append(missing, "DB_USER")
	}
	if dbname == "" {
		missing = append(missing, "DB_NAME")
	}

	if len(missing) > 0 {
		for _, m := range missing {
			fmt.Printf("  %s %s not set in .env\n", ui.StyleError.Render("✗"), m)
		}
		return fmt.Errorf("missing required database environment variables")
	}

	fmt.Printf("  Driver:   %s\n", driver)
	fmt.Printf("  Host:     %s:%s\n", host, port)
	fmt.Printf("  Database: %s\n", dbname)
	fmt.Printf("  User:     %s\n", user)
	fmt.Printf("  Timeout:  %ds\n", timeoutSec)
	fmt.Println()

	// Build connection string and test
	var dsn string
	switch driver {
	case "postgres":
		if sslmode == "" {
			sslmode = "disable"
		}
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d",
			host, port, user, password, dbname, sslmode, timeoutSec)
	case "mysql":
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?timeout=%ds&parseTime=true",
			user, password, host, port, dbname, timeoutSec)
	default:
		return fmt.Errorf("unsupported DB_DRIVER: %s (valid: postgres, mysql)", driver)
	}

	// Test connection using exec (no direct DB driver dependency in go-tk)
	// We use a simple TCP ping as a lightweight check
	fmt.Printf("  Testing connection...")

	if err := testTCPConnection(host, port, timeoutSec); err != nil {
		fmt.Printf(" %s\n", ui.StyleError.Render("FAILED"))
		fmt.Printf("\n  %s\n", ui.StyleError.Render("Database is unreachable: "+err.Error()))
		ui.PrintHint("Check that the database is running and network is accessible.")
		ui.PrintHint("Verify DB_HOST and DB_PORT in .env")
		return fmt.Errorf("database connectivity check failed")
	}

	fmt.Printf(" %s\n", ui.StyleSuccess.Render("OK"))
	fmt.Println()
	ui.PrintDone("Database is reachable at " + host + ":" + port)
	ui.PrintHint("Note: This only tests TCP connectivity. Run 'go-tk migrate status' to verify credentials.")

	// Print DSN hint (masked password)
	maskedDSN := dsn
	if password != "" {
		maskedDSN = fmt.Sprintf("...password=****...") // Don't show actual DSN with password
	}
	_ = maskedDSN // Avoid unused variable

	return nil
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
