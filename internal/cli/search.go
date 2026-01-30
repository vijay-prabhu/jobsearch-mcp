package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/config"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/database"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/output"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search conversations",
	Long: `Search across all conversations by company, recruiter, position, or email subject.

Examples:
  jobsearch search stripe
  jobsearch search "senior engineer"
  jobsearch search recruiting`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	query := strings.Join(args, " ")

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

	// Search
	results, err := db.Search(ctx, query)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		fmt.Printf("No conversations found matching: %s\n", query)
		return nil
	}

	fmt.Printf("Found %d conversation(s) matching: %s\n\n", len(results), query)

	// Output
	return output.Output(outputFmt, results)
}
