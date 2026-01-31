package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/config"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/database"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/output"
)

var filtersCmd = &cobra.Command{
	Use:   "filters",
	Short: "Manage learned email filters",
	Long: `Manage AI-learned and user-defined email filters.

The system learns from:
  - LLM classifications (suggests domains/keywords)
  - Your feedback (false positives/negatives)

Use subcommands to:
  - list: View all learned filters
  - approve: Approve an AI suggestion
  - reject: Reject/delete a filter
  - export: Export filters to add to config.toml`,
}

var filtersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all learned filters",
	RunE:  runFiltersList,
}

var filtersApproveCmd = &cobra.Command{
	Use:   "approve <filter-id>",
	Short: "Approve an AI-suggested filter",
	Args:  cobra.ExactArgs(1),
	RunE:  runFiltersApprove,
}

var filtersRejectCmd = &cobra.Command{
	Use:   "reject <filter-id>",
	Short: "Reject and delete a filter",
	Args:  cobra.ExactArgs(1),
	RunE:  runFiltersReject,
}

var filtersExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export confirmed filters for config.toml",
	Long: `Export all confirmed (user + ai_confirmed) filters in TOML format.

You can add this output to your config.toml [filters] section
to make the learned filters permanent.`,
	RunE: runFiltersExport,
}

var (
	filtersTypeFlag   string
	filtersSourceFlag string
)

func init() {
	rootCmd.AddCommand(filtersCmd)
	filtersCmd.AddCommand(filtersListCmd)
	filtersCmd.AddCommand(filtersApproveCmd)
	filtersCmd.AddCommand(filtersRejectCmd)
	filtersCmd.AddCommand(filtersExportCmd)

	filtersListCmd.Flags().StringVar(&filtersTypeFlag, "type", "", "Filter by type (domain_whitelist, domain_blacklist, subject_keyword, body_keyword, subject_blacklist)")
	filtersListCmd.Flags().StringVar(&filtersSourceFlag, "source", "", "Filter by source (user, ai_suggested, ai_confirmed)")
}

func runFiltersList(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	opts := database.LearnedFilterListOptions{}
	if filtersTypeFlag != "" {
		opts.FilterType = &filtersTypeFlag
	}
	if filtersSourceFlag != "" {
		opts.Source = &filtersSourceFlag
	}

	filters, err := db.ListLearnedFilters(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to list filters: %w", err)
	}

	if outputFmt == "json" {
		return output.JSON(filters)
	}

	if len(filters) == 0 {
		fmt.Println("No learned filters yet.")
		fmt.Println("\nFilters are learned from:")
		fmt.Println("  - LLM classifications during sync")
		fmt.Println("  - Your feedback (jobsearch feedback false-positive/false-negative)")
		return nil
	}

	// Group by source for display
	suggested := []database.LearnedFilter{}
	confirmed := []database.LearnedFilter{}

	for _, f := range filters {
		if f.Source == database.FilterSourceAISuggested {
			suggested = append(suggested, f)
		} else {
			confirmed = append(confirmed, f)
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	if len(suggested) > 0 {
		fmt.Println("AI SUGGESTIONS (pending approval):")
		fmt.Fprintln(w, "ID\tTYPE\tVALUE\tCONFIDENCE")
		fmt.Fprintln(w, "──\t────\t─────\t──────────")
		for _, f := range suggested {
			conf := "-"
			if f.Confidence != nil {
				conf = fmt.Sprintf("%.0f%%", *f.Confidence*100)
			}
			// Show short ID for easier use
			shortID := f.ID
			if len(shortID) > 8 {
				shortID = shortID[:8]
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", shortID, f.FilterType, f.Value, conf)
		}
		w.Flush()
		fmt.Println()
		fmt.Println("Use 'jobsearch filters approve <id>' to confirm a suggestion")
		fmt.Println("Use 'jobsearch filters reject <id>' to delete a suggestion")
		fmt.Println()
	}

	if len(confirmed) > 0 {
		fmt.Println("CONFIRMED FILTERS (active):")
		fmt.Fprintln(w, "ID\tTYPE\tVALUE\tSOURCE")
		fmt.Fprintln(w, "──\t────\t─────\t──────")
		for _, f := range confirmed {
			shortID := f.ID
			if len(shortID) > 8 {
				shortID = shortID[:8]
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", shortID, f.FilterType, f.Value, f.Source)
		}
		w.Flush()
		fmt.Println()
		fmt.Println("Use 'jobsearch filters export' to add these to your config.toml")
	}

	return nil
}

func runFiltersApprove(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	filterID := args[0]

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Try to find by full ID or prefix
	filters, err := db.ListLearnedFilters(ctx, database.LearnedFilterListOptions{})
	if err != nil {
		return err
	}

	var found *database.LearnedFilter
	for i := range filters {
		if filters[i].ID == filterID || strings.HasPrefix(filters[i].ID, filterID) {
			found = &filters[i]
			break
		}
	}

	if found == nil {
		return fmt.Errorf("filter not found: %s", filterID)
	}

	if err := db.ApproveLearnedFilter(ctx, found.ID); err != nil {
		return fmt.Errorf("failed to approve filter: %w", err)
	}

	fmt.Printf("Approved filter: %s = %s\n", found.FilterType, found.Value)
	fmt.Println("This filter is now active and will be used during sync.")

	return nil
}

func runFiltersReject(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	filterID := args[0]

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Try to find by full ID or prefix
	filters, err := db.ListLearnedFilters(ctx, database.LearnedFilterListOptions{})
	if err != nil {
		return err
	}

	var found *database.LearnedFilter
	for i := range filters {
		if filters[i].ID == filterID || strings.HasPrefix(filters[i].ID, filterID) {
			found = &filters[i]
			break
		}
	}

	if found == nil {
		return fmt.Errorf("filter not found: %s", filterID)
	}

	if err := db.DeleteLearnedFilter(ctx, found.ID); err != nil {
		return fmt.Errorf("failed to delete filter: %w", err)
	}

	fmt.Printf("Deleted filter: %s = %s\n", found.FilterType, found.Value)

	return nil
}

func runFiltersExport(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Get all confirmed filters grouped by type
	types := []string{
		database.FilterTypeDomainWhitelist,
		database.FilterTypeDomainBlacklist,
		database.FilterTypeSubjectBlacklist,
		database.FilterTypeSubjectKeyword,
		database.FilterTypeBodyKeyword,
	}

	fmt.Println("# Add these to your config.toml [filters] section:")
	fmt.Println()

	for _, filterType := range types {
		values, err := db.GetLearnedFiltersByType(ctx, filterType)
		if err != nil {
			continue
		}

		if len(values) == 0 {
			continue
		}

		// Convert filter type to TOML key
		tomlKey := filterType
		switch filterType {
		case database.FilterTypeDomainWhitelist:
			tomlKey = "domain_whitelist"
		case database.FilterTypeDomainBlacklist:
			tomlKey = "domain_blacklist"
		case database.FilterTypeSubjectBlacklist:
			tomlKey = "subject_blacklist"
		case database.FilterTypeSubjectKeyword:
			tomlKey = "subject_keywords"
		case database.FilterTypeBodyKeyword:
			tomlKey = "body_keywords"
		}

		fmt.Printf("# Learned %s\n", tomlKey)
		fmt.Printf("%s = [\n", tomlKey)
		for _, v := range values {
			fmt.Printf("    %q,\n", v)
		}
		fmt.Println("]")
		fmt.Println()
	}

	return nil
}
