package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/spf13/cobra"

	"github.com/octogate/octogate/internal/payment"
)

func NewSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search the web",
		Long:  `Search the web using the x402 Hub. Results are returned as JSON.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]

			// Read environment variables
			hubURL := os.Getenv("X402_HUB")
			if hubURL == "" {
				hubURL = "http://localhost:8080"
			}

			privKey := os.Getenv("EVM_PRIVATE_KEY")
			if privKey == "" {
				return fmt.Errorf("EVM_PRIVATE_KEY environment variable not set")
			}

			network := os.Getenv("NETWORK")
			if network == "" {
				network = "eip155:84532" // Base Sepolia default
			}

			// Create EIP-191 signer
			signer, err := payment.NewEIP191Signer(privKey)
			if err != nil {
				return fmt.Errorf("failed to create signer: %w", err)
			}

			// Construct search URL
			searchURL := fmt.Sprintf("%s/v1/search?q=%s", hubURL, url.QueryEscape(query))

			// Step 1: Make initial request without payment
			fmt.Fprintf(os.Stderr, "→ Requesting: %s\n", query)

			client := &http.Client{}
			req, err := http.NewRequest("GET", searchURL, nil)
			if err != nil {
				return fmt.Errorf("failed to create request: %w", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			// Read response body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}

			// Step 2: Handle 402 Payment Required
			if resp.StatusCode == http.StatusPaymentRequired {
				fmt.Fprintf(os.Stderr, "✓ Challenge received (402)\n")

				// Parse payment challenge
				var challengeMap map[string]interface{}
				if err := json.Unmarshal(body, &challengeMap); err != nil {
					return fmt.Errorf("failed to parse payment challenge: %w", err)
				}

				// Convert to PaymentChallenge struct
				challenge := &payment.PaymentChallenge{
					X402Version: int(challengeMap["x402Version"].(float64)),
					Price:       challengeMap["price"].(string),
					Currency:    challengeMap["currency"].(string),
					PayTo:       challengeMap["payTo"].(string),
					Nonce:       challengeMap["nonce"].(string),
				}

				if networks, ok := challengeMap["networks"].([]interface{}); ok {
					challenge.Networks = make([]string, len(networks))
					for i, n := range networks {
						challenge.Networks[i] = n.(string)
					}
				}

				// Step 3: Sign the challenge with EIP-191
				fmt.Fprintf(os.Stderr, "✓ Signing with EVM_PRIVATE_KEY\n")
				signature, err := signer.SignChallenge(challenge)
				if err != nil {
					return fmt.Errorf("failed to sign challenge: %w", err)
				}

				// Step 4: Retry request with signed payment
				fmt.Fprintf(os.Stderr, "✓ Submitting payment: %s USDC\n", challenge.Price)

				paymentHeader := signature.ToX402Header(network)
				req, err := http.NewRequest("GET", searchURL, nil)
				if err != nil {
					return fmt.Errorf("failed to create retry request: %w", err)
				}

				req.Header.Set("X-PAYMENT", paymentHeader)

				resp, err := client.Do(req)
				if err != nil {
					return fmt.Errorf("payment request failed: %w", err)
				}
				defer resp.Body.Close()

				// Read response body again
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return fmt.Errorf("failed to read response: %w", err)
				}

				// Step 5: Handle result
				if resp.StatusCode != http.StatusOK {
					return fmt.Errorf("payment failed: status %d", resp.StatusCode)
				}

				fmt.Fprintf(os.Stderr, "✓ Payment verified\n")
				fmt.Fprintf(os.Stderr, "✓ Results received\n\n")

				// Parse and pretty-print results
				var result map[string]interface{}
				if err := json.Unmarshal(body, &result); err != nil {
					return fmt.Errorf("failed to parse response: %w", err)
				}

				data, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal response: %w", err)
				}

				fmt.Println(string(data))
				return nil
			}

			// Handle success (payment not required)
			if resp.StatusCode == http.StatusOK {
				fmt.Fprintf(os.Stderr, "✓ Results received (no payment required)\n\n")

				var result map[string]interface{}
				if err := json.Unmarshal(body, &result); err != nil {
					return fmt.Errorf("failed to parse response: %w", err)
				}

				data, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal response: %w", err)
				}

				fmt.Println(string(data))
				return nil
			}

			// Unexpected status
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		},
	}
}
