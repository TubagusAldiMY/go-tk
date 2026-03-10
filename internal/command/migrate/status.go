package migrate

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/TubagusAldiMY/go-tk/internal/ui"
)

// MigrationInfo holds display state for one migration file.
type MigrationInfo struct {
	Version  uint
	Name     string
	Filename string
	Applied  bool
}

// GetStatus reads .up.sql files from migrationsDir and marks each as applied
// if its version number is <= currentVersion.
func GetStatus(migrationsDir string, currentVersion uint, dirty bool) ([]MigrationInfo, error) {
	entries, err := os.ReadDir(migrationsDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading migrations directory: %w", err)
	}

	var result []MigrationInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".up.sql") {
			continue
		}
		version, name, ok := parseMigrationFilename(e.Name())
		if !ok {
			continue
		}
		result = append(result, MigrationInfo{
			Version:  version,
			Name:     name,
			Filename: e.Name(),
			Applied:  version <= currentVersion,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Version < result[j].Version
	})
	return result, nil
}

// PrintStatusTable renders the migration status dashboard.
func PrintStatusTable(statuses []MigrationInfo, driver, dsn string, currentVersion uint, dirty bool) {
	fmt.Printf("\nMigration Status — %s @ %s\n", strings.ToUpper(driver), sanitizeForDisplay(dsn))
	fmt.Println(strings.Repeat("─", 60))

	if len(statuses) == 0 {
		fmt.Println("  " + ui.InfoMsg("No migration files found."))
		fmt.Println(strings.Repeat("─", 60))
		return
	}

	applied, pending := 0, 0
	for _, s := range statuses {
		if s.Applied {
			fmt.Printf("  %s  %s\n", ui.StyleSuccess.Render("✓"), s.Filename)
			applied++
		} else {
			fmt.Printf("  %s  %s  %s\n", ui.StyleMuted.Render("○"), s.Filename, ui.StyleWarning.Render("PENDING"))
			pending++
		}
	}

	if dirty {
		fmt.Printf("\n  %s\n", ui.StyleError.Render("⚠  Database is in a dirty state — manual intervention required"))
	}

	fmt.Println(strings.Repeat("─", 60))
	summary := fmt.Sprintf("%d applied · %d pending", applied, pending)
	if pending > 0 {
		summary += "  →  Run: go-tk migrate up"
	}
	fmt.Printf("  %s\n\n", ui.StyleMuted.Render(summary))
}

// parseMigrationFilename extracts version and name from "20260310_create_users.up.sql".
func parseMigrationFilename(filename string) (version uint, name string, ok bool) {
	base := strings.TrimSuffix(filename, ".up.sql")
	idx := strings.Index(base, "_")
	if idx < 0 {
		return 0, "", false
	}
	v, err := strconv.ParseUint(base[:idx], 10, 64)
	if err != nil {
		return 0, "", false
	}
	return uint(v), base[idx+1:], true
}

// sanitizeForDisplay removes credentials from a DSN for safe display.
func sanitizeForDisplay(dsn string) string {
	if idx := strings.Index(dsn, "@"); idx != -1 {
		return dsn[idx+1:]
	}
	return dsn
}
