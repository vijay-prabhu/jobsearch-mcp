package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vijay-prabhu/jobsearch-mcp/internal/config"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/database"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/output"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show job search statistics",
	Long: `Display aggregate statistics about your job search.

Examples:
  jobsearch stats             # Overall stats
  jobsearch stats --since=7d  # Stats for last 7 days
  jobsearch stats --detailed  # Detailed breakdown with charts`,
	RunE: runStats,
}

var (
	statsSince    string
	statsDetailed bool
)

func init() {
	rootCmd.AddCommand(statsCmd)
	statsCmd.Flags().StringVar(&statsSince, "since", "", "Time period (e.g., 7d, 2w, 1m)")
	statsCmd.Flags().BoolVar(&statsDetailed, "detailed", false, "Show detailed statistics with breakdowns")
}

func runStats(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

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

	// Parse time filter
	var since *time.Time
	if statsSince != "" {
		duration, err := parseDuration(statsSince)
		if err != nil {
			return fmt.Errorf("invalid duration: %w", err)
		}
		sinceTime := time.Now().Add(-duration)
		since = &sinceTime
	}

	// Get basic stats
	stats, err := db.GetStats(ctx, since)
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	if !statsDetailed {
		return output.Output(outputFmt, stats)
	}

	// Get detailed stats
	detailed, err := getDetailedStats(ctx, db, since)
	if err != nil {
		return fmt.Errorf("failed to get detailed stats: %w", err)
	}

	if outputFmt == "json" {
		return output.JSON(detailed)
	}

	// Print detailed text output
	printDetailedStats(detailed)
	return nil
}

// DetailedStats contains extended statistics
type DetailedStats struct {
	Basic           *database.Stats `json:"basic"`
	ByStatus        map[string]int  `json:"by_status"`
	ByCompany       []CompanyStat   `json:"by_company"`
	RecentActivity  []ActivityStat  `json:"recent_activity"`
	ResponseMetrics ResponseMetrics `json:"response_metrics"`
}

// CompanyStat shows statistics per company
type CompanyStat struct {
	Company    string `json:"company"`
	EmailCount int    `json:"email_count"`
	Status     string `json:"status"`
	DaysAgo    int    `json:"days_since_activity"`
}

// ActivityStat shows activity over time
type ActivityStat struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// ResponseMetrics shows response time analysis
type ResponseMetrics struct {
	AvgDaysToFirstResponse float64 `json:"avg_days_to_first_response"`
	ConversationsWithReply int     `json:"conversations_with_reply"`
	TotalInbound           int     `json:"total_inbound"`
	ResponseRate           float64 `json:"response_rate_percent"`
}

func getDetailedStats(ctx context.Context, db *database.DB, since *time.Time) (*DetailedStats, error) {
	// Get basic stats
	basic, err := db.GetStats(ctx, since)
	if err != nil {
		return nil, err
	}

	// Get all conversations
	opts := database.ListOptions{}
	if since != nil {
		opts.Since = since
	}
	convs, err := db.ListConversations(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Build status breakdown
	byStatus := make(map[string]int)
	for _, c := range convs {
		byStatus[string(c.Status)]++
	}

	// Build company breakdown (top 10)
	byCompany := make([]CompanyStat, 0, len(convs))
	for _, c := range convs {
		byCompany = append(byCompany, CompanyStat{
			Company:    c.Company,
			EmailCount: c.EmailCount,
			Status:     string(c.Status),
			DaysAgo:    c.DaysSinceActivity(),
		})
	}
	// Sort by email count descending
	sort.Slice(byCompany, func(i, j int) bool {
		return byCompany[i].EmailCount > byCompany[j].EmailCount
	})
	if len(byCompany) > 10 {
		byCompany = byCompany[:10]
	}

	// Build activity by day (last 14 days)
	activityByDay := make(map[string]int)
	for _, c := range convs {
		day := c.LastActivityAt.Format("2006-01-02")
		activityByDay[day]++
	}

	recentActivity := make([]ActivityStat, 0)
	for i := 13; i >= 0; i-- {
		day := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		recentActivity = append(recentActivity, ActivityStat{
			Date:  day,
			Count: activityByDay[day],
		})
	}

	// Calculate response metrics
	var inbound, withReply int
	for _, c := range convs {
		if c.Direction == database.DirectionInbound {
			inbound++
			if c.EmailCount > 1 {
				withReply++
			}
		}
	}

	responseRate := 0.0
	if inbound > 0 {
		responseRate = float64(withReply) / float64(inbound) * 100
	}

	return &DetailedStats{
		Basic:          basic,
		ByStatus:       byStatus,
		ByCompany:      byCompany,
		RecentActivity: recentActivity,
		ResponseMetrics: ResponseMetrics{
			ConversationsWithReply: withReply,
			TotalInbound:           inbound,
			ResponseRate:           responseRate,
		},
	}, nil
}

func printDetailedStats(d *DetailedStats) {
	// Header
	fmt.Println("Job Search Statistics (Detailed)")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println()

	// Summary
	fmt.Println("Summary")
	fmt.Println(strings.Repeat("-", 30))
	fmt.Printf("  Total Conversations: %d\n", d.Basic.TotalConversations)
	fmt.Printf("  Total Emails:        %d\n", d.Basic.TotalEmails)
	fmt.Printf("  Waiting on me:       %d\n", d.Basic.WaitingOnMe)
	fmt.Printf("  Waiting on them:     %d\n", d.Basic.WaitingOnThem)
	fmt.Printf("  Stale:               %d\n", d.Basic.Stale)
	fmt.Printf("  Closed:              %d\n", d.Basic.Closed)
	fmt.Println()

	// Response metrics
	fmt.Println("Response Metrics")
	fmt.Println(strings.Repeat("-", 30))
	fmt.Printf("  Inbound conversations:   %d\n", d.ResponseMetrics.TotalInbound)
	fmt.Printf("  Conversations with reply: %d\n", d.ResponseMetrics.ConversationsWithReply)
	fmt.Printf("  Response rate:           %.1f%%\n", d.ResponseMetrics.ResponseRate)
	fmt.Println()

	// Top companies
	if len(d.ByCompany) > 0 {
		fmt.Println("Top Companies (by email count)")
		fmt.Println(strings.Repeat("-", 30))
		for _, c := range d.ByCompany {
			statusIcon := statusToIcon(c.Status)
			fmt.Printf("  %s %-20s %d emails (%dd ago)\n", statusIcon, truncate(c.Company, 20), c.EmailCount, c.DaysAgo)
		}
		fmt.Println()
	}

	// Activity chart (ASCII)
	fmt.Println("Activity (Last 14 Days)")
	fmt.Println(strings.Repeat("-", 30))
	maxCount := 0
	for _, a := range d.RecentActivity {
		if a.Count > maxCount {
			maxCount = a.Count
		}
	}
	if maxCount > 0 {
		for _, a := range d.RecentActivity {
			bar := ""
			barLen := (a.Count * 20) / maxCount
			for i := 0; i < barLen; i++ {
				bar += "█"
			}
			dayLabel := a.Date[5:] // MM-DD
			fmt.Printf("  %s %s %d\n", dayLabel, bar, a.Count)
		}
	} else {
		fmt.Println("  No activity in the last 14 days")
	}
}

func statusToIcon(status string) string {
	switch status {
	case "waiting_on_me":
		return "📩"
	case "waiting_on_them":
		return "⏳"
	case "stale":
		return "⚠️"
	case "closed":
		return "✅"
	default:
		return "📋"
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
