package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/config"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/database"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/output"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List conversations",
	Long: `List job search conversations with optional filters.

Examples:
  jobsearch list                           # List all conversations
  jobsearch list --status=waiting_on_me    # List conversations needing your response
  jobsearch list --since=7d                # List conversations from last 7 days
  jobsearch list -o json                   # Output as JSON`,
	RunE: runList,
}

var (
	listStatus string
	listSince  string
	listLimit  int
)

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVar(&listStatus, "status", "", "Filter by status (waiting_on_me, waiting_on_them, stale, active, closed)")
	listCmd.Flags().StringVar(&listSince, "since", "", "Filter by time (e.g., 7d, 2w, 1m)")
	listCmd.Flags().IntVar(&listLimit, "limit", 0, "Maximum number of results")
}

func runList(cmd *cobra.Command, args []string) error {
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

	// Build query options
	opts := database.ListOptions{
		Limit: listLimit,
	}

	if listStatus != "" {
		status := database.ConversationStatus(listStatus)
		opts.Status = &status
	}

	if listSince != "" {
		since, err := parseDuration(listSince)
		if err != nil {
			return fmt.Errorf("invalid duration: %w", err)
		}
		sinceTime := time.Now().Add(-since)
		opts.Since = &sinceTime
	}

	// Query database
	convs, err := db.ListConversations(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to list conversations: %w", err)
	}

	// Output
	return output.Output(outputFmt, convs)
}

// parseDuration parses a human-readable duration like "7d", "2w", "1m"
func parseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration format")
	}

	unit := s[len(s)-1]
	valueStr := s[:len(s)-1]

	var value int
	if _, err := fmt.Sscanf(valueStr, "%d", &value); err != nil {
		return 0, fmt.Errorf("invalid duration value")
	}

	switch unit {
	case 'd':
		return time.Duration(value) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(value) * 7 * 24 * time.Hour, nil
	case 'm':
		return time.Duration(value) * 30 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown duration unit: %c (use d, w, or m)", unit)
	}
}
