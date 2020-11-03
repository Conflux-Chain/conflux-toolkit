package rpc

import (
	"fmt"

	"github.com/Conflux-Chain/conflux-toolkit/util"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/spf13/cobra"
)

func init() {
	accountCmd := &cobra.Command{
		Use:   "account",
		Short: "Get account info",
		Run:   getAccountInfo,
	}

	accountCmd.PersistentFlags().StringVar(&address, "address", "", "Account address in HEX format")
	accountCmd.MarkPersistentFlagRequired("address")

	rootCmd.AddCommand(accountCmd)
}

func getAccountInfo(cmd *cobra.Command, args []string) {
	client := MustCreateClient()
	defer client.Close()

	info, err := client.GetAccountInfo(types.Address(address))
	if err != nil {
		fmt.Println("Failed to get account info:", err.Error())
		return
	}

	prettyPrintAccount(info)
}

func prettyPrintAccount(info types.AccountInfo) {
	m := linkedhashmap.New()

	m.Put("balance", util.DisplayValueWithUnit(info.Balance.ToInt()))
	m.Put("nonce", info.Nonce.ToInt())
	m.Put("codeHash", info.CodeHash)
	m.Put("stakingBalance", util.DisplayValueWithUnit(info.StakingBalance.ToInt()))
	m.Put("collateralForStorage", util.DisplayValueWithUnit(info.CollateralForStorage.ToInt()))
	m.Put("accumulatedInterestReturn", util.DisplayValueWithUnit(info.AccumulatedInterestReturn.ToInt()))
	m.Put("admin", info.Admin)

	content, err := m.ToJSON()
	if err != nil {
		fmt.Println("Failed to marshal data to JSON:", err.Error())
	} else {
		fmt.Println(string(content))
	}
}
