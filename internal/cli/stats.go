package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/config"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/database"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/output"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show job search statistics",
	Long: `Display aggregate statistics about your job search.

Examples:
  jobsearch stats            # Overall stats
  jobsearch stats --since=7d # Stats for last 7 days`,
	RunE: runStats,
}

var statsSince string

func init() {
	rootCmd.AddCommand(statsCmd)
	statsCmd.Flags().StringVar(&statsSince, "since", "", "Time period (e.g., 7d, 2w, 1m)")
}

func runStats(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	// Open database
	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Parse time filter
	var since *time.Time
	if statsSince != "" {
		duration, err := parseDuration(statsSince)
		if err != nil {
			return fmt.Errorf("invalid duration: %w", err)
		}
		sinceTime := time.Now().Add(-duration)
		since = &sinceTime
	}

	// Get stats
	stats, err := db.GetStats(ctx, since)
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	// Output
	return output.Output(outputFmt, stats)
}
