package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/Conflux-Chain/conflux-toolkit/account"
	"github.com/Conflux-Chain/conflux-toolkit/contract"
	"github.com/Conflux-Chain/conflux-toolkit/converter"
	"github.com/Conflux-Chain/conflux-toolkit/rpc"
	"github.com/Conflux-Chain/conflux-toolkit/transfer"
	"github.com/Conflux-Chain/conflux-toolkit/util"
	"github.com/sirupsen/logrus"
)

var rootCmd = util.CreateUsageCommand("conflux-toolkit", "Conflux toolkit")

func init() {
	setLogLevel()
	account.SetParent(rootCmd)
	rpc.SetParent(rootCmd)
	contract.SetParent(rootCmd)
	transfer.SetParent(rootCmd)
	converter.SetParent(rootCmd)
}

func setLogLevel() {
	levelStr := os.Getenv("LOGLEVEL")
	level, err := strconv.ParseUint(levelStr, 10, 32)
	if err != nil {
		fmt.Println(err)
	}
	logrus.SetLevel(logrus.Level(level))
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
