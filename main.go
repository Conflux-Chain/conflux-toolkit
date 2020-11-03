package main

import (
	"fmt"
	"os"

	"github.com/Conflux-Chain/conflux-toolkit/account"
	"github.com/Conflux-Chain/conflux-toolkit/contract"
	"github.com/Conflux-Chain/conflux-toolkit/rpc"
	"github.com/Conflux-Chain/conflux-toolkit/util"
)

var rootCmd = util.CreateUsageCommand("conflux-toolkit", "Conflux toolkit")

func init() {
	account.SetParent(rootCmd)
	rpc.SetParent(rootCmd)
	contract.SetParent(rootCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
