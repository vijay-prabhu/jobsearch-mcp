package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/vijay-prabhu/jobsearch-mcp/internal/config"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/database"
	"github.com/vijay-prabhu/jobsearch-mcp/internal/mcp"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server (stdio transport)",
	Long: `Start the MCP (Model Context Protocol) server using stdio transport.

This allows AI assistants like Claude Desktop to interact with your job search data.

Add to Claude Desktop config (~/Library/Application Support/Claude/claude_desktop_config.json):

{
  "mcpServers": {
    "jobsearch": {
      "command": "/path/to/jobsearch",
      "args": ["mcp"]
    }
  }
}`,
	RunE: runMCP,
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}

func runMCP(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if MCP is enabled
	if !cfg.MCP.Enabled {
		return fmt.Errorf("MCP server is disabled in config")
	}

	// Open database
	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Create MCP server
	server := mcp.New(db, cfg)

	// Handle interrupt
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		cancel()
	}()

	// Run server
	return server.Start(ctx)
}
