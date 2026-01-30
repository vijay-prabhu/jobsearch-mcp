package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/classifier"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/config"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/database"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/email/gmail"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/filter"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/tracker"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Fetch and process new emails from Gmail",
	Long: `Sync fetches new emails from your Gmail account, filters them
for job-related content, and stores them in the local database.

On first run, it will open a browser for Google authentication.`,
	RunE: runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	// Ensure directories exist
	if err := cfg.EnsureDirectories(); err != nil {
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

	// Authenticate
	fmt.Println("Authenticating with Gmail...")
	if err := provider.Authenticate(ctx); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	userEmail, _ := provider.GetUserEmail(ctx)
	fmt.Printf("Authenticated as: %s\n", userEmail)

	// Initialize filter
	f := filter.New(cfg.Filters)

	// Initialize classifier client (optional)
	var classifierClient *classifier.Client
	classifierURL := cfg.ClassifierURL()
	classifierClient = classifier.New(classifierURL)

	if classifierClient.IsRunning(ctx) {
		fmt.Printf("Classification service: connected (%s)\n", classifierURL)
	} else {
		fmt.Println("Classification service: not running (skipping LLM classification)")
		classifierClient = nil
	}

	// Create tracker and run sync
	t := tracker.New(db, provider, f, classifierClient, cfg)

	fmt.Println()
	fmt.Println("Fetching emails...")

	result, err := t.Sync(ctx)
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	// Display results
	fmt.Println()
	fmt.Println("Sync complete:")
	fmt.Printf("  Emails fetched:        %d\n", result.EmailsFetched)
	fmt.Printf("  Job-related:           %d\n", result.EmailsFiltered)
	if result.EmailsClassified > 0 {
		fmt.Printf("  Classified by LLM:     %d\n", result.EmailsClassified)
	}
	fmt.Printf("  New conversations:     %d\n", result.ConversationsNew)
	fmt.Printf("  Updated conversations: %d\n", result.ConversationsUpdated)

	if len(result.Errors) > 0 {
		fmt.Println()
		fmt.Printf("Warnings: %d\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Printf("  - %v\n", e)
		}
	}

	// Show pending actions if any
	showPendingActions(ctx, db)

	return nil
}

func showPendingActions(ctx context.Context, db *database.DB) {
	status := database.StatusWaitingOnMe
	convs, err := db.ListConversations(ctx, database.ListOptions{
		Status: &status,
		Limit:  5,
	})
	if err != nil || len(convs) == 0 {
		return
	}

	fmt.Println()
	fmt.Printf("Action needed (%d conversations waiting on you):\n", len(convs))
	for _, c := range convs {
		recruiter := c.Company
		if c.RecruiterName != nil && *c.RecruiterName != "" {
			recruiter = *c.RecruiterName + " @ " + c.Company
		}
		fmt.Printf("  - %s (%d days)\n", recruiter, c.DaysSinceActivity())
	}
	fmt.Println()
	fmt.Println("Run 'jobsearch list --status=waiting_on_me' for details.")
}
