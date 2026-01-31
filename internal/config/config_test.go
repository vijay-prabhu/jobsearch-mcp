package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.Gmail.MaxResults != 100 {
		t.Errorf("expected MaxResults=100, got %d", cfg.Gmail.MaxResults)
	}

	if cfg.LLM.Primary != "ollama" {
		t.Errorf("expected Primary=ollama, got %s", cfg.LLM.Primary)
	}

	if cfg.Classifier.Port != 8642 {
		t.Errorf("expected Port=8642, got %d", cfg.Classifier.Port)
	}

	if cfg.Tracking.StaleAfterDays != 7 {
		t.Errorf("expected StaleAfterDays=7, got %d", cfg.Tracking.StaleAfterDays)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{
			name:    "valid default config",
			modify:  func(c *Config) {},
			wantErr: false,
		},
		{
			name: "invalid max_results",
			modify: func(c *Config) {
				c.Gmail.MaxResults = 0
			},
			wantErr: true,
		},
		{
			name: "invalid llm primary",
			modify: func(c *Config) {
				c.LLM.Primary = "invalid"
			},
			wantErr: true,
		},
		{
			name: "invalid classifier port",
			modify: func(c *Config) {
				c.Classifier.Port = 0
			},
			wantErr: true,
		},
		{
			name: "invalid mcp transport",
			modify: func(c *Config) {
				c.MCP.Transport = "http"
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.modify(cfg)

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{"~/test", filepath.Join(home, "test")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		result, err := expandPath(tt.input)
		if err != nil {
			t.Errorf("expandPath(%q) error: %v", tt.input, err)
		}
		if result != tt.expected {
			t.Errorf("expandPath(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestClassifierURL(t *testing.T) {
	cfg := Default()
	expected := "http://localhost:8642"

	if got := cfg.ClassifierURL(); got != expected {
		t.Errorf("ClassifierURL() = %q, want %q", got, expected)
	}
}

func TestStaleDuration(t *testing.T) {
	cfg := Default()
	expected := 7 * 24 * 60 * 60 // 7 days in seconds

	got := cfg.Tracking.StaleDuration().Seconds()
	if int(got) != expected {
		t.Errorf("StaleDuration() = %v seconds, want %v", got, expected)
	}
}
