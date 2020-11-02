package sponsorfaucet

import (
	"fmt"

	"github.com/Conflux-Chain/conflux-toolkit/util"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Query sponsor faucet info",
	Run: func(cmd *cobra.Command, args []string) {
		queryInfo()
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

func queryInfo() {
	totalGas := util.MustDecodeUint256(util.MustCall("0x5f929b7b"))
	fmt.Println("Total gas:", util.DisplayValueWithUnit(totalGas))

	totalCollateral := util.MustDecodeUint256(util.MustCall("0x5607a2c4"))
	fmt.Println("Total collateral:", util.DisplayValueWithUnit(totalCollateral))

	boundGas := util.MustDecodeUint256(util.MustCall("0x9ca64c41"))
	fmt.Println("Gas bound:", util.DisplayValueWithUnit(boundGas))

	boundCollateral := util.MustDecodeUint256(util.MustCall("0xca9c8c37"))
	fmt.Println("Collateral bound:", util.DisplayValueWithUnit(boundCollateral))

	boundFee := util.MustDecodeUint256(util.MustCall("0x85eb4dab"))
	fmt.Println("Fee bound:", util.DisplayValueWithUnit(boundFee))
}
