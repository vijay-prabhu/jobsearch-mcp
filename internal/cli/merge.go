package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vijay-prabhu/jobsearch-mcp/internal/config"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/database"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/output"
)

var mergeCmd = &cobra.Command{
	Use:   "merge <conv1> <conv2>",
	Short: "Merge two conversations into one",
	Long: `Merge two conversations into a single conversation.

The second conversation's emails will be moved to the first conversation,
and the second conversation will be deleted.

Arguments can be:
  - Company name (case-insensitive, partial match)
  - Conversation ID

Examples:
  jobsearch merge "Stripe" "stripe-2"
  jobsearch merge abc123 def456`,
	Args: cobra.ExactArgs(2),
	RunE: runMerge,
}

func init() {
	rootCmd.AddCommand(mergeCmd)
}

func runMerge(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	id1 := args[0]
	id2 := args[1]

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

	// Find first conversation
	conv1, err := findConversation(ctx, db, id1)
	if err != nil {
		return fmt.Errorf("failed to find first conversation: %w", err)
	}
	if conv1 == nil {
		return fmt.Errorf("conversation not found: %s", id1)
	}

	// Find second conversation
	conv2, err := findConversation(ctx, db, id2)
	if err != nil {
		return fmt.Errorf("failed to find second conversation: %w", err)
	}
	if conv2 == nil {
		return fmt.Errorf("conversation not found: %s", id2)
	}

	// Prevent merging into self
	if conv1.ID == conv2.ID {
		return fmt.Errorf("cannot merge a conversation with itself")
	}

	// Perform merge
	result, err := db.MergeConversations(ctx, conv1.ID, conv2.ID)
	if err != nil {
		return fmt.Errorf("merge failed: %w", err)
	}

	// Output
	if outputFmt == "json" {
		return output.JSON(result)
	}

	fmt.Printf("Merged conversations:\n")
	fmt.Printf("  From: %s (%s)\n", conv2.Company, conv2.ID)
	fmt.Printf("  Into: %s (%s)\n", conv1.Company, conv1.ID)
	fmt.Printf("  Emails moved: %d\n", result.EmailsMoved)
	fmt.Printf("  New total emails: %d\n", result.TotalEmails)

	return nil
}

// findConversation finds a conversation by company name or ID
func findConversation(ctx context.Context, db *database.DB, identifier string) (*database.Conversation, error) {
	// Try by company first
	conv, err := db.GetConversationByCompany(ctx, identifier)
	if err != nil {
		return nil, err
	}
	if conv != nil {
		return conv, nil
	}

	// Try by ID
	conv, err = db.GetConversation(ctx, identifier)
	if err != nil {
		return nil, err
	}
	if conv != nil {
		return conv, nil
	}

	// Try search and use first result
	results, err := db.Search(ctx, identifier)
	if err != nil {
		return nil, err
	}
	if len(results) > 0 {
		return &results[0], nil
	}

	return nil, nil
}
