package config

import "time"

// Config represents the application configuration
type Config struct {
	Gmail      GmailConfig      `toml:"gmail"`
	Database   DatabaseConfig   `toml:"database"`
	LLM        LLMConfig        `toml:"llm"`
	Classifier ClassifierConfig `toml:"classifier"`
	Filters    FilterConfig     `toml:"filters"`
	Tracking   TrackingConfig   `toml:"tracking"`
	Privacy    PrivacyConfig    `toml:"privacy"`
	MCP        MCPConfig        `toml:"mcp"`
}

// GmailConfig contains Gmail-specific settings
type GmailConfig struct {
	CredentialsPath string `toml:"credentials_path"`
	TokenPath       string `toml:"token_path"`
	MaxResults      int    `toml:"max_results"`
}

// DatabaseConfig contains database settings
type DatabaseConfig struct {
	Path string `toml:"path"`
}

// LLMConfig contains LLM provider settings
type LLMConfig struct {
	Primary  string       `toml:"primary"`
	Fallback string       `toml:"fallback"`
	Ollama   OllamaConfig `toml:"ollama"`
	OpenAI   OpenAIConfig `toml:"openai"`
}

// OllamaConfig contains Ollama-specific settings
type OllamaConfig struct {
	Model string `toml:"model"`
	Host  string `toml:"host"`
}

// OpenAIConfig contains OpenAI-specific settings
type OpenAIConfig struct {
	Model string `toml:"model"`
	// API key is read from OPENAI_API_KEY environment variable
}

// ClassifierConfig contains classification service settings
type ClassifierConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

// URL returns the full classifier service URL
func (c ClassifierConfig) URL() string {
	return c.Host + ":" + string(rune(c.Port))
}

// FilterConfig contains email filtering rules
type FilterConfig struct {
	DomainWhitelist  []string `toml:"domain_whitelist"`
	DomainBlacklist  []string `toml:"domain_blacklist"`
	SubjectBlacklist []string `toml:"subject_blacklist"`
	SubjectKeywords  []string `toml:"subject_keywords"`
	BodyKeywords     []string `toml:"body_keywords"`
}

// TrackingConfig contains conversation tracking settings
type TrackingConfig struct {
	StaleAfterDays int `toml:"stale_after_days"`
}

// StaleDuration returns the stale threshold as a duration
func (t TrackingConfig) StaleDuration() time.Duration {
	return time.Duration(t.StaleAfterDays) * 24 * time.Hour
}

// PrivacyConfig contains privacy-related settings
type PrivacyConfig struct {
	StoreEmailBody    bool   `toml:"store_email_body"`
	EncryptionKeyPath string `toml:"encryption_key_path"`
}

// MCPConfig contains MCP server settings
type MCPConfig struct {
	Enabled   bool   `toml:"enabled"`
	Transport string `toml:"transport"`
}

// Default returns a Config with sensible defaults
func Default() *Config {
	return &Config{
		Gmail: GmailConfig{
			CredentialsPath: "~/.config/jobsearch/credentials.json",
			TokenPath:       "~/.config/jobsearch/token.json",
			MaxResults:      100,
		},
		Database: DatabaseConfig{
			Path: "~/.local/share/jobsearch/jobsearch.db",
		},
		LLM: LLMConfig{
			Primary:  "ollama",
			Fallback: "openai",
			Ollama: OllamaConfig{
				Model: "llama3.2:1b",
				Host:  "http://localhost:11434",
			},
			OpenAI: OpenAIConfig{
				Model: "gpt-4o-mini",
			},
		},
		Classifier: ClassifierConfig{
			Host: "http://localhost",
			Port: 8642,
		},
		Filters: FilterConfig{
			DomainWhitelist: []string{
				"greenhouse.io",
				"lever.co",
				"ashbyhq.com",
				"smartrecruiters.com",
			},
			DomainBlacklist: []string{
				"noreply@linkedin.com",
				"mailchimp.com",
				"sendgrid.net",
			},
			SubjectBlacklist: []string{
				"job alert",
				"new jobs for you",
				"weekly digest",
			},
			SubjectKeywords: []string{
				"opportunity",
				"role",
				"position",
				"interview",
				"application",
				"candidate",
			},
			BodyKeywords: []string{
				"your background",
				"schedule a call",
				"interested in",
				"reaching out",
				"resume",
				"experience",
			},
		},
		Tracking: TrackingConfig{
			StaleAfterDays: 7,
		},
		Privacy: PrivacyConfig{
			StoreEmailBody:    false,
			EncryptionKeyPath: "~/.config/jobsearch/encryption.key",
		},
		MCP: MCPConfig{
			Enabled:   true,
			Transport: "stdio",
		},
	}
}
