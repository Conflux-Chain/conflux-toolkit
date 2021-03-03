package account

import (
	"github.com/Conflux-Chain/conflux-toolkit/util"
	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	"github.com/spf13/cobra"
)

var (
	am *sdk.AccountManager = sdk.NewAccountManager("keystore", util.MAINNET)

	rootCmd = util.CreateUsageCommand("account", "Account subcommand")
)

// SetParent sets parent command
func SetParent(parent *cobra.Command) {
	parent.AddCommand(rootCmd)
}
