package main

import (
	"os"

	"github.com/vijay-prabhu/jobsearch-mcp/internal/cli"
)

// Version information (set by build script)
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	cli.SetVersionInfo(Version, Commit, BuildTime)
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
