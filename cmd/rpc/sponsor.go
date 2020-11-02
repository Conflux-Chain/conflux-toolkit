package rpc

import (
	"fmt"

	"github.com/Conflux-Chain/conflux-toolkit/util"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/spf13/cobra"
)

var sponsorCmd = &cobra.Command{
	Use:   "sponsor",
	Short: "Get sponsor info for specified contract",
	Run: func(cmd *cobra.Command, args []string) {
		getSponsorInfo()
	},
}

func init() {
	sponsorCmd.PersistentFlags().StringVar(&address, "contract", "", "Contract address in HEX format")
	sponsorCmd.MarkPersistentFlagRequired("contract")

	rootCmd.AddCommand(sponsorCmd)
}

func getSponsorInfo() {
	client := util.MustGetClient()
	defer client.Close()

	info, err := client.GetSponsorInfo(types.Address(address))
	if err != nil {
		fmt.Println("Failed to get sponsor info:", err.Error())
		return
	}

	prettyPrintSponsor(info)
}

func prettyPrintSponsor(info types.SponsorInfo) {
	m := linkedhashmap.New()

	m.Put("gasSponsor", info.SponsorForGas)
	m.Put("gasBalance", util.DisplayValueWithUnit(info.SponsorBalanceForGas.ToInt()))
	m.Put("gasFeeBound", util.DisplayValueWithUnit(info.SponsorGasBound.ToInt()))
	m.Put("collateralSponsor", info.SponsorForCollateral)
	m.Put("collateralBalance", util.DisplayValueWithUnit(info.SponsorBalanceForCollateral.ToInt()))

	content, err := m.ToJSON()
	if err != nil {
		fmt.Println("Failed to marshal data to JSON:", err.Error())
	} else {
		fmt.Println(string(content))
	}
}
