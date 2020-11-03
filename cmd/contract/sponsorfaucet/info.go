package sponsorfaucet

import (
	"fmt"
	"math/big"
	"os"

	"github.com/Conflux-Chain/conflux-toolkit/util"
	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
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
	client := util.MustGetClient()
	defer client.Close()

	contract, err := client.GetContract([]byte(abiJSON), types.NewAddress(util.Contract))
	if err != nil {
		fmt.Println("Failed to read ABI:", err.Error())
		return
	}

	result := new(big.Int)

	mustCall(contract, &result, "gas_total_limit")
	fmt.Println("Total gas:", util.DisplayValueWithUnit(result))

	mustCall(contract, &result, "collateral_total_limit")
	fmt.Println("Total collateral:", util.DisplayValueWithUnit(result))

	mustCall(contract, &result, "gas_bound")
	fmt.Println("Gas bound:", util.DisplayValueWithUnit(result))

	mustCall(contract, &result, "collateral_bound")
	fmt.Println("Collateral bound:", util.DisplayValueWithUnit(result))

	mustCall(contract, &result, "upper_bound")
	fmt.Println("Fee bound:", util.DisplayValueWithUnit(result))
}

func mustCall(contract *sdk.Contract, resultPtr interface{}, method string, args ...interface{}) {
	if err := contract.Call(nil, resultPtr, method, args...); err != nil {
		fmt.Printf("Failed to call method %v: %v\n", method, err.Error())
		os.Exit(1)
	}
}
