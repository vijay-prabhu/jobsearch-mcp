package gmail

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"google.golang.org/api/gmail/v1"

	"github.com/vijay-prabhu/jobsearch-mcp/internal/email"
)

// buildQuery constructs a Gmail search query from FetchOptions
func buildQuery(opts email.FetchOptions) string {
	var parts []string

	// Add date filter
	if opts.After != nil {
		parts = append(parts, fmt.Sprintf("after:%s", opts.After.Format("2006/01/02")))
	}

	// Include both inbox and sent emails when IncludeSent is true
	// This ensures we capture replies to recruiters
	if opts.IncludeSent {
		parts = append(parts, "(in:inbox OR in:sent)")
	}

	// Add custom query if provided
	if opts.Query != "" {
		parts = append(parts, opts.Query)
	}

	return strings.Join(parts, " ")
}

// convertMessage converts a Gmail message to our Email type
func convertMessage(msg *gmail.Message) email.Email {
	e := email.Email{
		ID:       msg.Id,
		ThreadID: msg.ThreadId,
		Snippet:  msg.Snippet,
		Headers:  make(map[string]string),
	}

	// Extract headers
	for _, header := range msg.Payload.Headers {
		switch strings.ToLower(header.Name) {
		case "subject":
			e.Subject = header.Value
		case "from":
			e.From = email.ParseAddress(header.Value)
		case "to":
			e.To = email.ParseAddresses(header.Value)
		case "date":
			if t, err := parseDate(header.Value); err == nil {
				e.Date = t
			}
		default:
			// Store other potentially useful headers
			if isUsefulHeader(header.Name) {
				e.Headers[header.Name] = header.Value
			}
		}
	}

	// Fallback to internal timestamp if date parsing failed
	if e.Date.IsZero() {
		e.Date = time.Unix(msg.InternalDate/1000, 0)
	}

	// Extract labels
	e.Labels = msg.LabelIds

	// Check read status
	e.IsRead = !containsLabel(msg.LabelIds, "UNREAD")

	// Extract body
	e.Body = extractBody(msg.Payload)

	return e
}

// parseDate attempts to parse various date formats
func parseDate(s string) (time.Time, error) {
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 MST",
		"2 Jan 2006 15:04:05 -0700",
		"Mon, 02 Jan 2006 15:04:05 -0700 (MST)",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", s)
}

// extractBody extracts the email body from the message payload
func extractBody(payload *gmail.MessagePart) string {
	// Try to get plain text first, then HTML
	text := extractPartByMime(payload, "text/plain")
	if text != "" {
		return text
	}

	html := extractPartByMime(payload, "text/html")
	if html != "" {
		// Basic HTML to text conversion (for snippet purposes)
		return stripHTMLTags(html)
	}

	return ""
}

// extractPartByMime recursively finds a part with the given MIME type
func extractPartByMime(part *gmail.MessagePart, mimeType string) string {
	if part == nil {
		return ""
	}

	// Check if this part matches
	if strings.HasPrefix(part.MimeType, mimeType) {
		if part.Body != nil && part.Body.Data != "" {
			decoded, err := base64.URLEncoding.DecodeString(part.Body.Data)
			if err == nil {
				return string(decoded)
			}
		}
	}

	// Check nested parts
	for _, subpart := range part.Parts {
		if result := extractPartByMime(subpart, mimeType); result != "" {
			return result
		}
	}

	return ""
}

// stripHTMLTags removes HTML tags (basic implementation)
func stripHTMLTags(html string) string {
	var result strings.Builder
	inTag := false

	for _, r := range html {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			result.WriteRune(r)
		}
	}

	// Clean up whitespace
	text := result.String()
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\t", " ")

	// Collapse multiple spaces/newlines
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}

	return strings.TrimSpace(text)
}

// containsLabel checks if a label is present
func containsLabel(labels []string, label string) bool {
	for _, l := range labels {
		if l == label {
			return true
		}
	}
	return false
}

// isUsefulHeader returns true for headers we want to preserve
func isUsefulHeader(name string) bool {
	useful := map[string]bool{
		"message-id":  true,
		"in-reply-to": true,
		"references":  true,
		"reply-to":    true,
	}
	return useful[strings.ToLower(name)]
}
