package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create default configuration file",
	RunE:  runConfigInit,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current configuration",
	RunE:  runConfigShow,
}

func init() {
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".config", "jobsearch")
	dataDir := filepath.Join(home, ".local", "share", "jobsearch")

	// Create directories
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	configFile := filepath.Join(configDir, "config.toml")

	// Check if config already exists
	if _, err := os.Stat(configFile); err == nil {
		fmt.Printf("Config file already exists at %s\n", configFile)
		fmt.Println("Use 'jobsearch config show' to view current configuration")
		return nil
	}

	// Write default config
	if err := os.WriteFile(configFile, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Created config file at %s\n", configFile)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Set up Gmail API credentials (see README.md)")
	fmt.Printf("  2. Save credentials.json to %s/\n", configDir)
	fmt.Println("  3. Run 'jobsearch sync' to authenticate and fetch emails")
	fmt.Println()
	fmt.Println("For local LLM, ensure Ollama is running with llama3.2:1b model:")
	fmt.Println("  ollama pull llama3.2:1b")
	fmt.Println("  ollama serve")

	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No config file found. Run 'jobsearch config init' to create one.")
			return nil
		}
		return fmt.Errorf("failed to read config: %w", err)
	}

	fmt.Printf("# Config file: %s\n\n", configPath)
	fmt.Println(string(data))
	return nil
}

const defaultConfig = `# JobSearch MCP Configuration

[gmail]
credentials_path = "~/.config/jobsearch/credentials.json"
token_path = "~/.config/jobsearch/token.json"
max_results = 100  # emails per sync

[database]
path = "~/.local/share/jobsearch/jobsearch.db"

[llm]
primary = "ollama"
fallback = "openai"

[llm.ollama]
model = "llama3.2:1b"
host = "http://localhost:11434"

[llm.openai]
model = "gpt-4o-mini"
# API key read from OPENAI_API_KEY env var

[classifier]
host = "http://localhost"
port = 8642
cache_enabled = true     # Cache LLM classifications to avoid redundant calls
min_confidence = 0.5     # Minimum confidence threshold (0.0-1.0)

[filters]
# Layer 1: Always include emails from these domains
domain_whitelist = [
    "greenhouse.io",
    "lever.co",
    "ashbyhq.com",
    "smartrecruiters.com",
    "workday.com",
    "myworkdayjobs.com"
]

# Layer 2: Always exclude
domain_blacklist = [
    "noreply@linkedin.com",
    "messages-noreply@linkedin.com",
    "mailchimp.com",
    "sendgrid.net",
    "marketing@",
    "newsletter@"
]

subject_blacklist = [
    "job alert",
    "new jobs for you",
    "weekly digest",
    "unsubscribe",
    "your daily job matches"
]

# Layer 3: Keyword scoring
subject_keywords = [
    "opportunity",
    "role",
    "position",
    "interview",
    "application",
    "candidate",
    "regarding your",
    "following up"
]

body_keywords = [
    "your background",
    "schedule a call",
    "interested in",
    "reaching out",
    "resume",
    "experience",
    "team at",
    "hiring for"
]

[tracking]
stale_after_days = 7

[privacy]
store_email_body = false
encryption_key_path = "~/.config/jobsearch/encryption.key"

[mcp]
enabled = true
transport = "stdio"
`
