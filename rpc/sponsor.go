package rpc

import (
	"fmt"

	"github.com/Conflux-Chain/conflux-toolkit/util"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/Conflux-Chain/go-conflux-sdk/types/cfxaddress"
	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/spf13/cobra"
)

func init() {
	sponsorCmd := &cobra.Command{
		Use:   "sponsor",
		Short: "Get sponsor info for specified contract",
		Run:   getSponsorInfo,
	}

	sponsorCmd.PersistentFlags().StringVar(&address, "contract", "", "Contract address in HEX format")
	sponsorCmd.MarkPersistentFlagRequired("contract")

	rootCmd.AddCommand(sponsorCmd)
}

func getSponsorInfo(cmd *cobra.Command, args []string) {
	client := MustCreateClient()
	defer client.Close()

	info, err := client.GetSponsorInfo(cfxaddress.MustNew(address))
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
