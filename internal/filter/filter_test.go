package filter

import (
	"testing"

	"github.com/vijay-prabhu/jobsearch-mcp/internal/config"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/email"
)

func TestFilter_DomainWhitelist(t *testing.T) {
	cfg := config.FilterConfig{
		DomainWhitelist: []string{"greenhouse.io", "lever.co"},
	}
	f := New(cfg)

	tests := []struct {
		name     string
		from     string
		wantIncl bool
		wantLyr  Layer
	}{
		{
			name:     "whitelisted domain",
			from:     "recruiter@greenhouse.io",
			wantIncl: true,
			wantLyr:  LayerWhitelist,
		},
		{
			name:     "non-whitelisted domain",
			from:     "someone@example.com",
			wantIncl: false,
			wantLyr:  LayerRejected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &email.Email{
				From: email.Address{Email: tt.from},
			}
			result := f.Apply(e)

			if result.Include != tt.wantIncl {
				t.Errorf("Include = %v, want %v", result.Include, tt.wantIncl)
			}
			if result.Layer != tt.wantLyr {
				t.Errorf("Layer = %v, want %v", result.Layer, tt.wantLyr)
			}
		})
	}
}

func TestFilter_DomainBlacklist(t *testing.T) {
	cfg := config.FilterConfig{
		DomainBlacklist: []string{"noreply@linkedin.com", "mailchimp.com"},
	}
	f := New(cfg)

	tests := []struct {
		name     string
		from     string
		wantIncl bool
		wantLyr  Layer
	}{
		{
			name:     "blacklisted email",
			from:     "noreply@linkedin.com",
			wantIncl: false,
			wantLyr:  LayerBlacklist,
		},
		{
			name:     "blacklisted domain",
			from:     "campaigns@mailchimp.com",
			wantIncl: false,
			wantLyr:  LayerBlacklist,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &email.Email{
				From: email.Address{Email: tt.from},
			}
			result := f.Apply(e)

			if result.Include != tt.wantIncl {
				t.Errorf("Include = %v, want %v", result.Include, tt.wantIncl)
			}
			if result.Layer != tt.wantLyr {
				t.Errorf("Layer = %v, want %v", result.Layer, tt.wantLyr)
			}
		})
	}
}

func TestFilter_SubjectBlacklist(t *testing.T) {
	cfg := config.FilterConfig{
		SubjectBlacklist: []string{"job alert", "weekly digest"},
	}
	f := New(cfg)

	tests := []struct {
		name     string
		subject  string
		wantIncl bool
		wantLyr  Layer
	}{
		{
			name:     "blacklisted subject",
			subject:  "Your weekly job alert",
			wantIncl: false,
			wantLyr:  LayerBlacklist,
		},
		{
			name:     "clean subject",
			subject:  "Exciting opportunity at Google",
			wantIncl: false,
			wantLyr:  LayerRejected, // No keywords to match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &email.Email{
				From:    email.Address{Email: "recruiter@company.com"},
				Subject: tt.subject,
			}
			result := f.Apply(e)

			if result.Include != tt.wantIncl {
				t.Errorf("Include = %v, want %v", result.Include, tt.wantIncl)
			}
			if result.Layer != tt.wantLyr {
				t.Errorf("Layer = %v, want %v", result.Layer, tt.wantLyr)
			}
		})
	}
}

func TestFilter_KeywordScoring(t *testing.T) {
	cfg := config.FilterConfig{
		SubjectKeywords: []string{"opportunity", "role", "position"},
		BodyKeywords:    []string{"your background", "schedule a call"},
	}
	f := New(cfg)

	tests := []struct {
		name     string
		subject  string
		body     string
		wantIncl bool
	}{
		{
			name:     "high keyword match",
			subject:  "Exciting opportunity for a senior role",
			body:     "Based on your background, I'd like to schedule a call",
			wantIncl: true,
		},
		{
			name:     "no keywords",
			subject:  "Hello there",
			body:     "Just checking in",
			wantIncl: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &email.Email{
				From:    email.Address{Email: "recruiter@company.com"},
				Subject: tt.subject,
				Body:    tt.body,
			}
			result := f.Apply(e)

			if result.Include != tt.wantIncl {
				t.Errorf("Include = %v, want %v (confidence: %.2f)", result.Include, tt.wantIncl, result.Confidence)
			}
		})
	}
}

func TestExtractCompanyFromDomain(t *testing.T) {
	tests := []struct {
		domain   string
		expected string
	}{
		{"stripe.com", "Stripe"},
		{"jobs.lever.co", ""}, // ATS domain (lever)
		{"greenhouse.io", ""}, // ATS domain
		{"mail.google.com", "Google"},
		{"acme.io", "Acme"},
	}

	for _, tt := range tests {
		result := ExtractCompanyFromDomain(tt.domain)
		if result != tt.expected {
			t.Errorf("ExtractCompanyFromDomain(%q) = %q, want %q", tt.domain, result, tt.expected)
		}
	}
}

func TestContainsWord(t *testing.T) {
	tests := []struct {
		text     string
		word     string
		expected bool
	}{
		{"this is a position", "position", true},
		{"preposition is not position", "position", true},
		{"hello world", "position", false},
		{"schedule a call with me", "schedule a call", true},
	}

	for _, tt := range tests {
		result := containsWord(tt.text, tt.word)
		if result != tt.expected {
			t.Errorf("containsWord(%q, %q) = %v, want %v", tt.text, tt.word, result, tt.expected)
		}
	}
}
