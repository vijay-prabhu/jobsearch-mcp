package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vijay-prabhu/jobsearch-mcp/internal/config"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/database"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/output"
)

var archiveCmd = &cobra.Command{
	Use:   "archive <company>",
	Short: "Archive a conversation",
	Long: `Archive a conversation to hide it from default list output.

Archived conversations are excluded from 'list' by default but can
be viewed with 'list --include-archived'.

Arguments can be:
  - Company name (case-insensitive, partial match)
  - Conversation ID

Examples:
  jobsearch archive "Stripe"
  jobsearch archive abc123`,
	Args: cobra.ExactArgs(1),
	RunE: runArchive,
}

var unarchiveCmd = &cobra.Command{
	Use:   "unarchive <company>",
	Short: "Unarchive a conversation",
	Long: `Unarchive a previously archived conversation.

This makes the conversation visible in default list output again.

Arguments can be:
  - Company name (case-insensitive, partial match)
  - Conversation ID

Examples:
  jobsearch unarchive "Stripe"
  jobsearch unarchive abc123`,
	Args: cobra.ExactArgs(1),
	RunE: runUnarchive,
}

func init() {
	rootCmd.AddCommand(archiveCmd)
	rootCmd.AddCommand(unarchiveCmd)
}

func runArchive(cmd *cobra.Command, args []string) error {
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

	// Find conversation
	conv, err := findConversation(ctx, db, identifier)
	if err != nil {
		return fmt.Errorf("failed to find conversation: %w", err)
	}
	if conv == nil {
		return fmt.Errorf("conversation not found: %s", identifier)
	}

	// Archive it
	result, err := db.ArchiveConversation(ctx, conv.ID)
	if err != nil {
		return fmt.Errorf("failed to archive: %w", err)
	}

	// Output
	if outputFmt == "json" {
		return output.JSON(result)
	}

	fmt.Printf("Archived: %s (%s)\n", result.Company, result.ConversationID)
	fmt.Println("Use 'jobsearch list --include-archived' to view archived conversations.")
	return nil
}

func runUnarchive(cmd *cobra.Command, args []string) error {
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

	// Find conversation (need to search including archived)
	conv, err := findConversationIncludingArchived(ctx, db, identifier)
	if err != nil {
		return fmt.Errorf("failed to find conversation: %w", err)
	}
	if conv == nil {
		return fmt.Errorf("conversation not found: %s", identifier)
	}

	// Unarchive it
	result, err := db.UnarchiveConversation(ctx, conv.ID)
	if err != nil {
		return fmt.Errorf("failed to unarchive: %w", err)
	}

	// Output
	if outputFmt == "json" {
		return output.JSON(result)
	}

	fmt.Printf("Unarchived: %s (%s)\n", result.Company, result.ConversationID)
	return nil
}

// findConversationIncludingArchived finds a conversation including archived ones
func findConversationIncludingArchived(ctx context.Context, db *database.DB, identifier string) (*database.Conversation, error) {
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
