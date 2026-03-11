package main

import (
	"fmt"
	"os"

	"github.com/octogate/octogate/cmd/x402/commands"
	"github.com/spf13/cobra"
)

var (
	Version = "0.1.0-alpha"
	Commit  = "dev"
)

func main() {
	root := &cobra.Command{
		Use:   "x402",
		Short: "x402 CLI — Pay-per-call API access for AI agents",
		Long: `x402 is a CLI tool that enables AI agents to access premium APIs
through a single binary. No API keys, no signups. Just one command.

Examples:
  x402 search "rust concurrency" --json
  x402 crawl https://example.com --json
  x402 balance
  x402 tools --category search`,
		Version: fmt.Sprintf("x402 %s (%s)", Version, Commit),
	}

	// Add subcommands
	root.AddCommand(
		commands.NewSearchCmd(),
		commands.NewCrawlCmd(),
		commands.NewScrapeCmd(),
		commands.NewEnrichCmd(),
		commands.NewBalanceCmd(),
		commands.NewToolsCmd(),
		commands.NewLoginCmd(),
		commands.NewStatusCmd(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
