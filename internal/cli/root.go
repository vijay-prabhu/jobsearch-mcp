package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	// Version info set from main
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"

	// Global flags
	configPath string
	outputFmt  string
)

// SetVersionInfo sets version information from build flags
func SetVersionInfo(v, c, b string) {
	version = v
	commit = c
	buildTime = b
}

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "jobsearch",
	Short: "A developer-focused job search automation tool",
	Long: `jobsearch-mcp tracks recruiter conversations from your email,
helping you stay organized during your job search.

It provides:
  - Email integration with Gmail (more providers coming)
  - Smart filtering and LLM-powered classification
  - Conversation tracking with status management
  - MCP server for AI assistant integration`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "",
		"config file (default: ~/.config/jobsearch/config.toml)")
	rootCmd.PersistentFlags().StringVarP(&outputFmt, "output", "o", "table",
		"output format (table, json)")

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
}

func initConfig() {
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding home directory: %v\n", err)
			os.Exit(1)
		}
		configPath = filepath.Join(home, ".config", "jobsearch", "config.toml")
	}
}

// versionCmd shows version information
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("jobsearch-mcp %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built:  %s\n", buildTime)
	},
}
