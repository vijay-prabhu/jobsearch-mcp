package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/vijay-prabhu/jobsearch-mcp/internal/config"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/database"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export conversations to CSV or JSON",
	Long: `Export job search conversations to a file.

Supported formats:
  - csv: Comma-separated values (spreadsheet-compatible)
  - json: JSON array of conversation objects

Examples:
  jobsearch export --format=csv > conversations.csv
  jobsearch export --format=json > conversations.json
  jobsearch export --format=csv --include-archived > all.csv`,
	RunE: runExport,
}

var (
	exportFormat          string
	exportIncludeArchived bool
)

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVar(&exportFormat, "format", "csv", "Export format (csv, json)")
	exportCmd.Flags().BoolVar(&exportIncludeArchived, "include-archived", false, "Include archived conversations")
}

func runExport(cmd *cobra.Command, args []string) error {
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

	// Get all conversations
	opts := database.ListOptions{
		IncludeArchived: exportIncludeArchived,
	}
	convs, err := db.ListConversations(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to list conversations: %w", err)
	}

	switch exportFormat {
	case "csv":
		return exportCSV(convs)
	case "json":
		return exportJSON(convs)
	default:
		return fmt.Errorf("unknown format: %s (use csv or json)", exportFormat)
	}
}

// ExportRow represents a row in the export (with additional computed fields)
type ExportRow struct {
	ID             string `json:"id"`
	Company        string `json:"company"`
	Position       string `json:"position"`
	RecruiterName  string `json:"recruiter_name"`
	RecruiterEmail string `json:"recruiter_email"`
	Direction      string `json:"direction"`
	Status         string `json:"status"`
	LastActivityAt string `json:"last_activity_at"`
	DaysSince      int    `json:"days_since_activity"`
	EmailCount     int    `json:"email_count"`
	Archived       bool   `json:"archived"`
	CreatedAt      string `json:"created_at"`
}

func toExportRow(c database.Conversation) ExportRow {
	row := ExportRow{
		ID:             c.ID,
		Company:        c.Company,
		Direction:      string(c.Direction),
		Status:         string(c.Status),
		LastActivityAt: c.LastActivityAt.Format(time.RFC3339),
		DaysSince:      c.DaysSinceActivity(),
		EmailCount:     c.EmailCount,
		Archived:       c.Archived,
		CreatedAt:      c.CreatedAt.Format(time.RFC3339),
	}
	if c.Position != nil {
		row.Position = *c.Position
	}
	if c.RecruiterName != nil {
		row.RecruiterName = *c.RecruiterName
	}
	if c.RecruiterEmail != nil {
		row.RecruiterEmail = *c.RecruiterEmail
	}
	return row
}

func exportCSV(convs []database.Conversation) error {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	// Write header
	header := []string{
		"id", "company", "position", "recruiter_name", "recruiter_email",
		"direction", "status", "last_activity_at", "days_since_activity",
		"email_count", "archived", "created_at",
	}
	if err := w.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write rows
	for _, c := range convs {
		row := toExportRow(c)
		record := []string{
			row.ID,
			row.Company,
			row.Position,
			row.RecruiterName,
			row.RecruiterEmail,
			row.Direction,
			row.Status,
			row.LastActivityAt,
			fmt.Sprintf("%d", row.DaysSince),
			fmt.Sprintf("%d", row.EmailCount),
			fmt.Sprintf("%t", row.Archived),
			row.CreatedAt,
		}
		if err := w.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

func exportJSON(convs []database.Conversation) error {
	rows := make([]ExportRow, len(convs))
	for i, c := range convs {
		rows[i] = toExportRow(c)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(rows); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	return nil
}
