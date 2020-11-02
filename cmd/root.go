package cmd

import (
	"fmt"
	"os"

	"github.com/Conflux-Chain/conflux-toolkit/cmd/account"
	"github.com/Conflux-Chain/conflux-toolkit/cmd/contract"
	"github.com/Conflux-Chain/conflux-toolkit/cmd/rpc"
	"github.com/Conflux-Chain/conflux-toolkit/util"
)

var rootCmd = util.CreateUsageCommand("conflux-toolkit", "Conflux toolkit")

func init() {
	rootCmd.PersistentFlags().StringVar(&util.URL, "url", "http://main.confluxrpc.org", "Conflux RPC URL")

	account.SetParent(rootCmd)
	rpc.SetParent(rootCmd)
	contract.SetParent(rootCmd)
}

// Execute is the command line entrypoint.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
