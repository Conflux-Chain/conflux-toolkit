package sponsorfaucet

import (
	"github.com/Conflux-Chain/conflux-toolkit/util"
	"github.com/spf13/cobra"
)

var rootCmd = util.CreateUsageCommand("sponsorfaucet", "Sponsor faucet subcommand")

// SetParent sets parent command
func SetParent(parent *cobra.Command) {
	parent.AddCommand(rootCmd)
}
