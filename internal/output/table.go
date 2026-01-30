package output

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/vijay-prabhu/jobsearch-mcp/internal/database"
)

// Table writes data as a formatted table to stdout
func Table(data interface{}) error {
	return TableTo(os.Stdout, data)
}

// TableTo writes data as a formatted table to the given writer
func TableTo(w io.Writer, data interface{}) error {
	switch v := data.(type) {
	case []database.Conversation:
		return conversationsTable(w, v)
	case *database.Conversation:
		return conversationDetail(w, v)
	case *database.Stats:
		return statsTable(w, v)
	default:
		return fmt.Errorf("unsupported data type for table output: %T", data)
	}
}

func conversationsTable(w io.Writer, convs []database.Conversation) error {
	if len(convs) == 0 {
		fmt.Fprintln(w, "No conversations found.")
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "COMPANY\tRECRUITER\tSTATUS\tEMAILS\tLAST ACTIVITY")
	fmt.Fprintln(tw, "-------\t---------\t------\t------\t-------------")

	for _, c := range convs {
		recruiter := ""
		if c.RecruiterName != nil && *c.RecruiterName != "" {
			recruiter = *c.RecruiterName
		} else if c.RecruiterEmail != nil {
			recruiter = *c.RecruiterEmail
		}

		status := formatStatusShort(c.Status)
		days := c.DaysSinceActivity()
		lastActivity := formatLastActivity(days)

		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\n",
			truncate(c.Company, 20),
			truncate(recruiter, 25),
			status,
			c.EmailCount,
			lastActivity,
		)
	}

	return tw.Flush()
}

func conversationDetail(w io.Writer, c *database.Conversation) error {
	fmt.Fprintf(w, "Company:     %s\n", c.Company)

	if c.RecruiterName != nil && *c.RecruiterName != "" {
		fmt.Fprintf(w, "Recruiter:   %s", *c.RecruiterName)
		if c.RecruiterEmail != nil {
			fmt.Fprintf(w, " <%s>", *c.RecruiterEmail)
		}
		fmt.Fprintln(w)
	} else if c.RecruiterEmail != nil {
		fmt.Fprintf(w, "Recruiter:   %s\n", *c.RecruiterEmail)
	}

	if c.Position != nil && *c.Position != "" {
		fmt.Fprintf(w, "Position:    %s\n", *c.Position)
	}

	fmt.Fprintf(w, "Status:      %s (%d days)\n", formatStatusShort(c.Status), c.DaysSinceActivity())
	fmt.Fprintf(w, "Emails:      %d\n", c.EmailCount)
	fmt.Fprintf(w, "Direction:   %s\n", c.Direction)
	fmt.Fprintf(w, "Created:     %s\n", c.CreatedAt.Format("Jan 02, 2006"))

	return nil
}

// ConversationWithEmails prints a conversation with its email timeline
func ConversationWithEmails(w io.Writer, c *database.Conversation, emails []database.Email, myEmail string) error {
	if err := conversationDetail(w, c); err != nil {
		return err
	}

	if len(emails) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Timeline:")

		for i, e := range emails {
			direction := "[IN] "
			if strings.EqualFold(e.FromAddress, myEmail) {
				direction = "[OUT]"
			}

			date := e.Date.Format("Jan 02")
			subject := ""
			if e.Subject != nil {
				subject = truncate(*e.Subject, 50)
			}

			marker := ""
			if i == len(emails)-1 {
				marker = " <- latest"
			}

			fmt.Fprintf(w, "  %s  %s  %s%s\n", date, direction, subject, marker)
		}
	}

	return nil
}

func statsTable(w io.Writer, s *database.Stats) error {
	fmt.Fprintln(w, "Job Search Statistics")
	fmt.Fprintln(w, strings.Repeat("-", 30))
	fmt.Fprintf(w, "Total conversations:    %d\n", s.TotalConversations)
	fmt.Fprintf(w, "Waiting on me:          %d\n", s.WaitingOnMe)
	fmt.Fprintf(w, "Waiting on them:        %d\n", s.WaitingOnThem)
	fmt.Fprintf(w, "Stale:                  %d\n", s.Stale)
	fmt.Fprintf(w, "Closed:                 %d\n", s.Closed)
	fmt.Fprintf(w, "Total emails:           %d\n", s.TotalEmails)

	if s.ResponseRate > 0 {
		fmt.Fprintf(w, "Response rate:          %.1f%%\n", s.ResponseRate*100)
	}
	if s.AvgResponseTime > 0 {
		fmt.Fprintf(w, "Avg response time:      %.1f days\n", s.AvgResponseTime)
	}

	return nil
}

func formatStatusShort(status database.ConversationStatus) string {
	switch status {
	case database.StatusWaitingOnMe:
		return "waiting_on_me"
	case database.StatusWaitingOnThem:
		return "waiting_on_them"
	case database.StatusStale:
		return "stale"
	case database.StatusClosed:
		return "closed"
	case database.StatusActive:
		return "active"
	default:
		return string(status)
	}
}

func formatLastActivity(days int) string {
	switch {
	case days == 0:
		return "today"
	case days == 1:
		return "yesterday"
	case days < 7:
		return fmt.Sprintf("%d days ago", days)
	case days < 30:
		return fmt.Sprintf("%d weeks ago", days/7)
	default:
		return fmt.Sprintf("%d days ago", days)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
