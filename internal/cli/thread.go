package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/config"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/database"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/email/gmail"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/filter"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/output"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/tracker"
)

var threadCmd = &cobra.Command{
	Use:   "thread <company|id>",
	Short: "Show full email thread with content",
	Long: `Fetch and display the full email thread for a conversation.

This command retrieves the complete email content from your email provider,
allowing you to read the entire conversation.

The identifier can be:
  - Company name (case-insensitive, partial match)
  - Conversation ID

Examples:
  jobsearch thread stripe
  jobsearch thread "Google Cloud"
  jobsearch thread stripe -o json`,
	Args: cobra.ExactArgs(1),
	RunE: runThread,
}

func init() {
	rootCmd.AddCommand(threadCmd)
}

func runThread(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	identifier := args[0]

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

	// Initialize Gmail provider
	provider := gmail.New(cfg.Gmail.CredentialsPath, cfg.Gmail.TokenPath)

	// Check if already authenticated
	if !provider.IsAuthenticated() {
		fmt.Println("Authenticating with Gmail...")
		if err := provider.Authenticate(ctx); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	// Create tracker (minimal setup, just need provider)
	f := filter.New(cfg.Filters)
	t := tracker.New(db, provider, f, nil, cfg)

	// Fetch the thread
	if outputFmt != "json" {
		fmt.Printf("Fetching thread for '%s'...\n\n", identifier)
	}

	thread, err := t.FetchThread(ctx, identifier)
	if err != nil {
		return err
	}

	// Output
	return output.Output(outputFmt, thread)
}
