package airdrop

import "github.com/spf13/cobra"

var airdropListFile string
var airdropNumber int
var from string

// AddAirdropListFileVar 。。。
func AddAirdropListFileVar(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&airdropListFile, "list", "", "airdrop list file path")
	cmd.MarkPersistentFlagRequired("airdrop")
}

// AddAirdropNumberVar ...
func AddAirdropNumberVar(cmd *cobra.Command) {
	cmd.PersistentFlags().IntVar(&airdropNumber, "number", 0, "airdrop number, the unit is CFX, 1 representds 1 CFX")
	cmd.MarkPersistentFlagRequired("airdrop")
}

// AddFromVar ...
func AddFromVar(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&from, "from", "", "airdop sender address")
	cmd.MarkPersistentFlagRequired("airdrop")
}
