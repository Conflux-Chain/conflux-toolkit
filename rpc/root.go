package rpc

import (
	"github.com/Conflux-Chain/conflux-toolkit/util"
	"github.com/spf13/cobra"
)

var (
	address string
	hash    string
	data    string

	rootCmd = util.CreateUsageCommand("rpc", "RPC subcommand")
)

func init() {
	AddURLVar(rootCmd)
}

// SetParent sets parent command
func SetParent(parent *cobra.Command) {
	parent.AddCommand(rootCmd)
}
