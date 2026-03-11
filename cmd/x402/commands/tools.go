package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewToolsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tools",
		Short: "List available tools from the Bazaar",
		Long:  `List all available tools and their prices from the x402 Hub Bazaar.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Available tools: TODO")
			return nil
		},
	}
}
