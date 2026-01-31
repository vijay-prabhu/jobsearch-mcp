package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/vijay-prabhu/jobsearch-mcp/internal/config"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/database"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/output"
)

var showCmd = &cobra.Command{
	Use:   "show <company|id>",
	Short: "Show conversation details",
	Long: `Show detailed information about a specific conversation.

The identifier can be:
  - Company name (case-insensitive, partial match)
  - Conversation ID

Examples:
  jobsearch show stripe
  jobsearch show "Google Cloud"`,
	Args: cobra.ExactArgs(1),
	RunE: runShow,
}

func init() {
	rootCmd.AddCommand(showCmd)
}

func runShow(cmd *cobra.Command, args []string) error {
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

	// Try to find by company first, then by ID, then by search
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

	// If still not found, try search and use first result
	if conv == nil {
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

	// Get emails for the conversation
	emails, err := db.ListEmailsForConversation(ctx, conv.ID)
	if err != nil {
		return fmt.Errorf("failed to get emails: %w", err)
	}

	// Get user email for direction detection
	userEmail := "" // We don't have it stored, but we can infer from outbound emails
	for _, e := range emails {
		if e.Direction == database.DirectionOutbound {
			userEmail = e.FromAddress
			break
		}
	}

	// Output
	if outputFmt == "json" {
		data := struct {
			Conversation *database.Conversation `json:"conversation"`
			Emails       []database.Email       `json:"emails"`
		}{
			Conversation: conv,
			Emails:       emails,
		}
		return output.JSON(data)
	}

	return output.ConversationWithEmails(os.Stdout, conv, emails, userEmail)
}
