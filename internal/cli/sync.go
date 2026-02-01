package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/vijay-prabhu/jobsearch-mcp/internal/classifier"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/config"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/database"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/email/gmail"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/filter"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/tracker"
)

var (
	syncDays       int
	syncFull       bool
	syncNoClassify bool
	syncBackground bool
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Fetch and process new emails from Gmail",
	Long: `Sync fetches new emails from your Gmail account, filters them
for job-related content, and stores them in the local database.

On first run, it will open a browser for Google authentication.

Examples:
  jobsearch sync              # Incremental sync (since last sync, or 30 days)
  jobsearch sync --days=60    # Fetch last 60 days
  jobsearch sync --full       # Full sync (ignore last sync time)
  jobsearch sync --no-classify # Skip LLM classification (faster, domain-filter only)
  jobsearch sync --background  # Quick sync with background classification`,
	RunE: runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().IntVar(&syncDays, "days", 0, "Number of days to fetch (default: 30, or since last sync)")
	syncCmd.Flags().BoolVar(&syncFull, "full", false, "Ignore last sync time and fetch from scratch")
	syncCmd.Flags().BoolVar(&syncNoClassify, "no-classify", false, "Skip LLM classification (faster, uses domain filtering only)")
	syncCmd.Flags().BoolVar(&syncBackground, "background", false, "Quick sync: show results immediately, classify in background")
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

	// Load learned filters from database
	loadLearnedFilters(ctx, db, f)

	// Initialize classifier client (optional)
	var classifierClient *classifier.Client
	if syncNoClassify {
		fmt.Println("Classification service: skipped (--no-classify flag)")
	} else {
		classifierURL := cfg.ClassifierURL()
		classifierClient = classifier.New(classifierURL)

		if classifierClient.IsRunning(ctx) {
			fmt.Printf("Classification service: connected (%s)\n", classifierURL)
		} else {
			fmt.Println("Classification service: not running (using domain filtering only)")
			fmt.Println("  Tip: Start classifier with 'make serve-classifier' for better accuracy")
			classifierClient = nil
		}
	}

	// Create tracker and run sync
	t := tracker.New(db, provider, f, classifierClient, cfg)

	// Build sync options
	syncOpts := tracker.SyncOptions{
		Days:               syncDays,
		FullSync:           syncFull,
		SkipClassification: syncNoClassify,
		BackgroundClassify: syncBackground,
	}

	fmt.Println()
	if syncDays > 0 {
		fmt.Printf("Syncing emails (last %d days)...\n", syncDays)
	} else if syncFull {
		fmt.Println("Syncing emails (full sync)...")
	} else {
		fmt.Println("Syncing emails...")
	}

	// Set up progress callback with terminal utilities
	var lastPhase tracker.ProgressPhase
	var phaseStartTime time.Time
	terminal := NewTerminal()

	syncOpts.Progress = func(p tracker.Progress) {
		// Track phase start time for ETA
		if p.Phase != lastPhase {
			phaseStartTime = time.Now()
		}
		p.StartedAt = phaseStartTime

		// Clear the line if we're in a terminal
		terminal.ClearLine()

		// Get phase color
		phaseColor := PhaseColor(string(p.Phase))

		// Format progress message
		var msg string
		var eta string
		switch p.Phase {
		case tracker.PhaseListingEmails:
			if p.Total > 0 {
				msg = fmt.Sprintf("📋 Listing emails: %d found", p.Current)
			} else {
				spinner := terminal.Spinner()
				msg = fmt.Sprintf("%s 📋 Listing emails...", spinner)
			}
		case tracker.PhaseFetchingEmails:
			pct := p.Percentage()
			if etaDur := p.ETA(); etaDur > 0 {
				eta = fmt.Sprintf(" (ETA: %s)", FormatETA(etaDur))
			}
			msg = fmt.Sprintf("📥 Downloading: %d/%d emails (%d%%)%s", p.Current, p.Total, pct, eta)
		case tracker.PhaseFiltering:
			spinner := terminal.Spinner()
			msg = fmt.Sprintf("%s 🔍 Filtering: %d emails", spinner, p.Total)
		case tracker.PhaseClassifying:
			pct := p.Percentage()
			if etaDur := p.ETA(); etaDur > 0 {
				eta = fmt.Sprintf(" (ETA: %s)", FormatETA(etaDur))
			}
			msg = fmt.Sprintf("🤖 Classifying with LLM: %d/%d (%d%%)%s", p.Current, p.Total, pct, eta)
		case tracker.PhaseValidating:
			pct := p.Percentage()
			if etaDur := p.ETA(); etaDur > 0 {
				eta = fmt.Sprintf(" (ETA: %s)", FormatETA(etaDur))
			}
			msg = fmt.Sprintf("🔬 Validating uncertain: %d/%d (%d%%)%s", p.Current, p.Total, pct, eta)
		case tracker.PhaseProcessing:
			pct := p.Percentage()
			if etaDur := p.ETA(); etaDur > 0 {
				eta = fmt.Sprintf(" (ETA: %s)", FormatETA(etaDur))
			}
			msg = fmt.Sprintf("💾 Processing: %d/%d emails (%d%%)%s", p.Current, p.Total, pct, eta)
		case tracker.PhaseUpdatingStatus:
			spinner := terminal.Spinner()
			msg = fmt.Sprintf("%s 🔄 Updating conversation statuses...", spinner)
		}

		// Apply color in terminal mode
		if terminal.UseColor {
			msg = terminal.Color(phaseColor, msg)
		}

		if terminal.IsTerminal {
			fmt.Print(msg)
			terminal.Flush()
		} else {
			// For non-terminals, print on phase change or every 10 items for long phases
			shouldPrint := p.Phase != lastPhase
			if p.Phase == tracker.PhaseClassifying || p.Phase == tracker.PhaseValidating {
				shouldPrint = shouldPrint || p.Current%10 == 0 || p.Current == p.Total
			}
			if shouldPrint {
				fmt.Println(msg)
			}
		}
		lastPhase = p.Phase
	}

	result, err := t.SyncWithOptions(ctx, syncOpts)

	// Clear progress line
	terminal.ClearLine()

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
	if result.EmailsPendingClassify > 0 {
		fmt.Printf("  Pending classification: %d (run sync again to classify)\n", result.EmailsPendingClassify)
	}
	fmt.Printf("  New conversations:     %d\n", result.ConversationsNew)
	fmt.Printf("  Updated conversations: %d\n", result.ConversationsUpdated)

	if result.ClassificationSkipped {
		fmt.Println()
		fmt.Println("  Note: LLM classification was skipped for faster sync.")
		fmt.Println("  Run 'jobsearch sync' (without --no-classify or --background) to classify pending emails.")
	}

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

// loadLearnedFilters loads confirmed filters from the database and adds them to the filter
func loadLearnedFilters(ctx context.Context, db *database.DB, f *filter.Filter) {
	filterTypes := []string{
		database.FilterTypeDomainWhitelist,
		database.FilterTypeDomainBlacklist,
		database.FilterTypeSubjectBlacklist,
		database.FilterTypeSubjectKeyword,
		database.FilterTypeBodyKeyword,
	}

	for _, filterType := range filterTypes {
		values, err := db.GetLearnedFiltersByType(ctx, filterType)
		if err != nil || len(values) == 0 {
			continue
		}
		f.AddLearnedFilters(filterType, values)
	}
}
