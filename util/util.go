package util

import (
	"fmt"
	"math/big"

	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
)

// CreateUsageCommand creates a command to display help.
func CreateUsageCommand(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
}

// DisplayValueWithUnit returns the display format for given drip value.
func DisplayValueWithUnit(drip *big.Int) string {
	if big.NewInt(1_000).Cmp(drip) > 0 {
		return fmt.Sprintf("%v Drip", drip)
	}

	if big.NewInt(1_000_000).Cmp(drip) > 0 {
		return fmt.Sprintf("%v Kdrip", decimal.NewFromBigInt(drip, -3))
	}

	if big.NewInt(1_000_000_000).Cmp(drip) > 0 {
		return fmt.Sprintf("%v Mdrip", decimal.NewFromBigInt(drip, -6))
	}

	if big.NewInt(1_000_000_000_000).Cmp(drip) > 0 {
		return fmt.Sprintf("%v Gdrip", decimal.NewFromBigInt(drip, -9))
	}

	return fmt.Sprintf("%v CFX", decimal.NewFromBigInt(drip, -18))
}
