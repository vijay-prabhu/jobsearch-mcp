package email

import (
	"strings"
	"time"
)

// Email represents a provider-agnostic email message
type Email struct {
	ID       string            // Provider-specific ID
	ThreadID string            // Thread/conversation ID
	Subject  string            // Email subject
	From     Address           // Sender address
	To       []Address         // Recipient addresses
	Date     time.Time         // Send/receive date
	Snippet  string            // Short preview text
	Body     string            // Full body (may be empty for privacy)
	Labels   []string          // Provider-specific labels
	IsRead   bool              // Read status
	Headers  map[string]string // Selected headers
}

// Address represents an email address with optional name
type Address struct {
	Name  string
	Email string
}

// String returns the formatted address
func (a Address) String() string {
	if a.Name == "" {
		return a.Email
	}
	return a.Name + " <" + a.Email + ">"
}

// Domain extracts the domain from the email address
func (a Address) Domain() string {
	parts := strings.Split(a.Email, "@")
	if len(parts) != 2 {
		return ""
	}
	return strings.ToLower(parts[1])
}

// Domain returns the sender's email domain
func (e *Email) Domain() string {
	return e.From.Domain()
}

// IsFromMe checks if this email was sent by the given address
func (e *Email) IsFromMe(myEmail string) bool {
	return strings.EqualFold(e.From.Email, myEmail)
}

// Direction returns "outbound" if sent by myEmail, "inbound" otherwise
func (e *Email) Direction(myEmail string) string {
	if e.IsFromMe(myEmail) {
		return "outbound"
	}
	return "inbound"
}

// ParseAddress parses an email address string like "Name <email@example.com>"
func ParseAddress(s string) Address {
	s = strings.TrimSpace(s)

	// Try to extract name and email from "Name <email>" format
	if start := strings.Index(s, "<"); start != -1 {
		if end := strings.Index(s, ">"); end > start {
			return Address{
				Name:  strings.TrimSpace(s[:start]),
				Email: strings.TrimSpace(s[start+1 : end]),
			}
		}
	}

	// Just an email address
	return Address{Email: s}
}

// ParseAddresses parses a comma-separated list of addresses
func ParseAddresses(s string) []Address {
	if s == "" {
		return nil
	}

	var addresses []Address
	for _, part := range strings.Split(s, ",") {
		if addr := ParseAddress(part); addr.Email != "" {
			addresses = append(addresses, addr)
		}
	}
	return addresses
}
