package contract

import (
	"github.com/Conflux-Chain/conflux-toolkit/cmd/contract/sponsorfaucet"
	"github.com/Conflux-Chain/conflux-toolkit/util"
	"github.com/spf13/cobra"
)

var rootCmd = util.CreateUsageCommand("contract", "Contract subcommand")

func init() {
	rootCmd.PersistentFlags().StringVar(&util.Contract, "contract", "", "Contract address in HEX format")
	rootCmd.MarkPersistentFlagRequired("contract")

	sponsorfaucet.SetParent(rootCmd)
}

// SetParent sets parent command
func SetParent(parent *cobra.Command) {
	parent.AddCommand(rootCmd)
}
