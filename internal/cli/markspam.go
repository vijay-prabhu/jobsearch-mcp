package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vijay-prabhu/jobsearch-mcp/internal/config"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/database"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/output"
)

var markSpamCmd = &cobra.Command{
	Use:   "mark-spam <company>",
	Short: "Mark a conversation as not job-related (false positive)",
	Long: `Mark a conversation as a false positive (not actually job-related).

This command:
1. Adds the recruiter's domain to your learned blacklist
2. Increments the false positive count for that domain
3. Archives the conversation

After multiple false positives from the same domain (default: 3),
the domain will be auto-blacklisted for future syncs.

Arguments can be:
  - Company name (case-insensitive, partial match)
  - Conversation ID

Examples:
  jobsearch mark-spam "Walmart"
  jobsearch mark-spam abc123`,
	Args: cobra.ExactArgs(1),
	RunE: runMarkSpam,
}

// MarkSpamResult contains the result of marking as spam
type MarkSpamResult struct {
	ConversationID     string `json:"conversation_id"`
	Company            string `json:"company"`
	Domain             string `json:"domain"`
	FalsePositiveCount int    `json:"false_positive_count"`
	AutoBlacklisted    bool   `json:"auto_blacklisted"`
	Archived           bool   `json:"archived"`
}

func init() {
	rootCmd.AddCommand(markSpamCmd)
}

func runMarkSpam(cmd *cobra.Command, args []string) error {
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
	conv, err := findConversationIncludingArchived(ctx, db, identifier)
	if err != nil {
		return fmt.Errorf("failed to find conversation: %w", err)
	}
	if conv == nil {
		return fmt.Errorf("conversation not found: %s", identifier)
	}

	// Extract domain from recruiter email
	var domain string
	if conv.RecruiterEmail != nil && *conv.RecruiterEmail != "" {
		parts := strings.Split(*conv.RecruiterEmail, "@")
		if len(parts) == 2 {
			domain = strings.ToLower(parts[1])
		}
	}

	result := &MarkSpamResult{
		ConversationID: conv.ID,
		Company:        conv.Company,
		Domain:         domain,
		Archived:       true,
	}

	// Mark the domain as false positive (increments counter)
	if domain != "" {
		if err := db.MarkFalsePositive(ctx, domain); err != nil {
			return fmt.Errorf("failed to record false positive: %w", err)
		}

		// Get the updated count
		count, err := db.GetFalsePositiveCount(ctx, domain)
		if err != nil {
			return fmt.Errorf("failed to get false positive count: %w", err)
		}
		result.FalsePositiveCount = count

		// Check if we should auto-blacklist (threshold is 3)
		autoBlacklistThreshold := 3
		if count >= autoBlacklistThreshold {
			if err := db.PromoteToAutoBlacklist(ctx, domain); err != nil {
				return fmt.Errorf("failed to auto-blacklist domain: %w", err)
			}
			result.AutoBlacklisted = true
		}
	}

	// Archive the conversation
	if _, err := db.ArchiveConversation(ctx, conv.ID); err != nil {
		return fmt.Errorf("failed to archive conversation: %w", err)
	}

	// Output
	if outputFmt == "json" {
		return output.JSON(result)
	}

	fmt.Printf("Marked as spam: %s\n", result.Company)
	if result.Domain != "" {
		fmt.Printf("  Domain: %s (false positive count: %d)\n", result.Domain, result.FalsePositiveCount)
		if result.AutoBlacklisted {
			fmt.Printf("  Domain auto-blacklisted (reached threshold)\n")
		}
	}
	fmt.Println("  Conversation archived")
	fmt.Println("\nEmails from this domain will be less likely to appear in future syncs.")
	return nil
}
