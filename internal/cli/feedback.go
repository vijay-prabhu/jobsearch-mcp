package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vijay-prabhu/jobsearch-mcp/internal/config"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/database"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/tracker"
)

var feedbackCmd = &cobra.Command{
	Use:   "feedback",
	Short: "Provide feedback on classifications",
	Long: `Provide feedback on email classifications to improve filtering.

Use subcommands to mark conversations as:
  - false-positive: Wrongly included (not job-related)
  - false-negative: Wrongly excluded (was job-related)`,
}

var feedbackFalsePositiveCmd = &cobra.Command{
	Use:   "false-positive <company-or-id>",
	Short: "Mark a conversation as incorrectly included",
	Long: `Mark a conversation as a false positive (it was included but shouldn't have been).

This will:
  - Add the sender's domain to the blacklist (as user filter)
  - Close the conversation
  - Help improve future filtering`,
	Args: cobra.ExactArgs(1),
	RunE: runFeedbackFalsePositive,
}

var feedbackFalseNegativeCmd = &cobra.Command{
	Use:   "false-negative <from-email>",
	Short: "Record a missed job-related email",
	Long: `Record that an email was incorrectly excluded (false negative).

This will:
  - Add the sender's domain to the whitelist (as user filter)
  - Help improve future filtering

Provide the sender's email address to learn from.`,
	Args: cobra.ExactArgs(1),
	RunE: runFeedbackFalseNegative,
}

func init() {
	rootCmd.AddCommand(feedbackCmd)
	feedbackCmd.AddCommand(feedbackFalsePositiveCmd)
	feedbackCmd.AddCommand(feedbackFalseNegativeCmd)
}

func runFeedbackFalsePositive(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	identifier := args[0]

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Find conversation
	conv, err := db.GetConversationByCompany(ctx, identifier)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if conv == nil {
		conv, err = db.GetConversation(ctx, identifier)
		if err != nil {
			return fmt.Errorf("database error: %w", err)
		}
	}
	if conv == nil {
		// Try search
		results, err := db.Search(ctx, identifier)
		if err != nil {
			return fmt.Errorf("search error: %w", err)
		}
		if len(results) > 0 {
			conv = &results[0]
		}
	}
	if conv == nil {
		return fmt.Errorf("conversation not found: %s", identifier)
	}

	// Create tracker with learner
	t := tracker.New(db, nil, nil, nil, cfg)

	if err := t.MarkFalsePositive(ctx, conv.ID); err != nil {
		return fmt.Errorf("failed to mark false positive: %w", err)
	}

	fmt.Printf("Marked '%s' as false positive.\n", conv.Company)
	fmt.Println("Filters have been updated based on this feedback.")
	fmt.Println("\nRun 'jobsearch filters list' to see suggested filters.")

	return nil
}

func runFeedbackFalseNegative(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	fromEmail := args[0]

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Create tracker with learner
	t := tracker.New(db, nil, nil, nil, cfg)

	if err := t.MarkFalseNegative(ctx, fromEmail, ""); err != nil {
		return fmt.Errorf("failed to record false negative: %w", err)
	}

	fmt.Printf("Recorded false negative for: %s\n", fromEmail)
	fmt.Println("Domain has been added to the whitelist.")
	fmt.Println("\nRun 'jobsearch filters list' to see all filters.")

	return nil
}
