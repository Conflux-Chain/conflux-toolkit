package sponsorfaucet

import (
	"fmt"
	"math/big"

	"github.com/Conflux-Chain/conflux-toolkit/contract/common"
	"github.com/Conflux-Chain/conflux-toolkit/util"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "info",
		Short: "Query sponsor faucet info",
		Run:   queryInfo,
	})
}

func queryInfo(cmd *cobra.Command, args []string) {
	contract := common.MustCreateContract(abiJSON)
	defer contract.Client.Close()

	var result *big.Int

	common.MustCall(contract, &result, "gas_total_limit")
	fmt.Println("Total gas:", util.DisplayValueWithUnit(result))

	common.MustCall(contract, &result, "collateral_total_limit")
	fmt.Println("Total collateral:", util.DisplayValueWithUnit(result))

	common.MustCall(contract, &result, "gas_bound")
	fmt.Println("Gas bound:", util.DisplayValueWithUnit(result))

	common.MustCall(contract, &result, "collateral_bound")
	fmt.Println("Collateral bound:", util.DisplayValueWithUnit(result))

	common.MustCall(contract, &result, "upper_bound")
	fmt.Println("Fee bound:", util.DisplayValueWithUnit(result))
}
