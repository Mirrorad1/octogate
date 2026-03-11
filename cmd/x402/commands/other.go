package commands

import (
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

func NewCrawlCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "crawl <url>",
		Short: "Crawl a URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Crawling %s: TODO\n", args[0])
			return nil
		},
	}
}

func NewScrapeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scrape <url>",
		Short: "Scrape a URL with selector",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Scraping %s: TODO\n", args[0])
			return nil
		},
	}
}

func NewEnrichCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enrich <domain>",
		Short: "Enrich domain data",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Enriching %s: TODO\n", args[0])
			return nil
		},
	}
}

func NewLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Login and setup wallet",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Login: TODO")
			return nil
		},
	}
}

func NewStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check Hub and wallet status",
		RunE: func(cmd *cobra.Command, args []string) error {
			hubURL := os.Getenv("X402_HUB")
			if hubURL == "" {
				hubURL = "http://localhost:8080"
			}

			privKey := os.Getenv("EVM_PRIVATE_KEY")
			network := os.Getenv("NETWORK")
			if network == "" {
				network = "eip155:84532" // Base Sepolia default
			}

			// Check Hub health
			resp, err := http.Get(hubURL + "/health")
			var hubStatus string
			if err != nil {
				hubStatus = "❌ " + err.Error()
			} else {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					hubStatus = "✓ Healthy"
				} else {
					hubStatus = fmt.Sprintf("❌ Status %d", resp.StatusCode)
				}
			}

			// Extract wallet address from private key
			var walletAddr string
			if privKey == "" {
				walletAddr = "❌ Not configured (EVM_PRIVATE_KEY not set)"
			} else {
				// For now, show that key is configured but don't derive address
				// In production, we'd derive the address from the private key
				walletAddr = "✓ Configured (EVM_PRIVATE_KEY set)"
			}

			// Output status
			fmt.Printf("x402 Status\n")
			fmt.Printf("===========\n\n")
			fmt.Printf("Hub:     %s\n", hubURL)
			fmt.Printf("Hub Status: %s\n", hubStatus)
			fmt.Printf("Network: %s\n", network)
			fmt.Printf("Wallet:  %s\n", walletAddr)

			return nil
		},
	}
}
