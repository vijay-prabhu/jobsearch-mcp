package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	// Expand path
	expandedPath, err := expandPath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to expand config path: %w", err)
	}

	// Read file
	data, err := os.ReadFile(expandedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s (run 'jobsearch config init' to create)", expandedPath)
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// Parse TOML
	cfg := Default()
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Expand paths in config
	if err := cfg.expandPaths(); err != nil {
		return nil, fmt.Errorf("failed to expand paths: %w", err)
	}

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

// MustLoad loads config or exits with error
func MustLoad(path string) *Config {
	cfg, err := Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return cfg
}

// expandPath expands ~ to home directory
func expandPath(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, path[1:]), nil
}

// expandPaths expands ~ in all path fields
func (c *Config) expandPaths() error {
	var err error

	c.Gmail.CredentialsPath, err = expandPath(c.Gmail.CredentialsPath)
	if err != nil {
		return err
	}

	c.Gmail.TokenPath, err = expandPath(c.Gmail.TokenPath)
	if err != nil {
		return err
	}

	c.Database.Path, err = expandPath(c.Database.Path)
	if err != nil {
		return err
	}

	c.Privacy.EncryptionKeyPath, err = expandPath(c.Privacy.EncryptionKeyPath)
	if err != nil {
		return err
	}

	return nil
}

// Validate checks that the configuration is valid
func (c *Config) Validate() error {
	var errs []error

	// Gmail validation
	if c.Gmail.CredentialsPath == "" {
		errs = append(errs, errors.New("gmail.credentials_path is required"))
	}
	if c.Gmail.TokenPath == "" {
		errs = append(errs, errors.New("gmail.token_path is required"))
	}
	if c.Gmail.MaxResults < 1 || c.Gmail.MaxResults > 5000 {
		errs = append(errs, errors.New("gmail.max_results must be between 1 and 5000"))
	}

	// Database validation
	if c.Database.Path == "" {
		errs = append(errs, errors.New("database.path is required"))
	}

	// LLM validation
	validProviders := map[string]bool{"ollama": true, "openai": true}
	if !validProviders[c.LLM.Primary] {
		errs = append(errs, fmt.Errorf("llm.primary must be 'ollama' or 'openai', got '%s'", c.LLM.Primary))
	}
	if c.LLM.Fallback != "" && !validProviders[c.LLM.Fallback] {
		errs = append(errs, fmt.Errorf("llm.fallback must be 'ollama' or 'openai', got '%s'", c.LLM.Fallback))
	}

	// Classifier validation
	if c.Classifier.Port < 1 || c.Classifier.Port > 65535 {
		errs = append(errs, errors.New("classifier.port must be between 1 and 65535"))
	}

	// Tracking validation
	if c.Tracking.StaleAfterDays < 1 {
		errs = append(errs, errors.New("tracking.stale_after_days must be at least 1"))
	}

	// MCP validation
	if c.MCP.Transport != "stdio" {
		errs = append(errs, fmt.Errorf("mcp.transport must be 'stdio', got '%s'", c.MCP.Transport))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// ClassifierURL returns the full URL for the classifier service
func (c *Config) ClassifierURL() string {
	return fmt.Sprintf("%s:%d", c.Classifier.Host, c.Classifier.Port)
}

// EnsureDirectories creates necessary directories for database and config
func (c *Config) EnsureDirectories() error {
	dirs := []string{
		filepath.Dir(c.Database.Path),
		filepath.Dir(c.Gmail.TokenPath),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}
