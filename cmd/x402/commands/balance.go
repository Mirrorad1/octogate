package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewBalanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "balance",
		Short: "Check USDC balance",
		Long:  `Check your USDC balance on Base or Solana.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Balance: TODO")
			return nil
		},
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "deposit",
			Short: "Show deposit address",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Println("Deposit address: TODO")
				return nil
			},
		},
		&cobra.Command{
			Use:   "withdraw <address> <amount>",
			Short: "Withdraw USDC",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Printf("Withdrawing %s to %s: TODO\n", args[1], args[0])
				return nil
			},
		},
	)

	return cmd
}
