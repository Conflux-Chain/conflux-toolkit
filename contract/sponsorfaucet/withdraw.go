package sponsorfaucet

import (
	"fmt"
	"os"

	"github.com/Conflux-Chain/conflux-toolkit/account"
	"github.com/Conflux-Chain/conflux-toolkit/contract/common"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/spf13/cobra"
)

var recipient string

func init() {
	withdrawCmd := &cobra.Command{
		Use:   "withdraw",
		Short: "Withdraw funds from sponsor faucet (owner only)",
		Run:   withdraw,
	}

	account.AddAccountVar(withdrawCmd)
	account.AddGasPriceVar(withdrawCmd)

	withdrawCmd.PersistentFlags().StringVar(&recipient, "recipient", "", "Recipient for withdrawal, empty for owner")

	withdrawCmd.PersistentFlags().StringVar(&account.ValueStr, "amount", "", "Amount to withdraw in CFX")
	withdrawCmd.MarkPersistentFlagRequired("amount")

	rootCmd.AddCommand(withdrawCmd)
}

func withdraw(cmd *cobra.Command, args []string) {
	from := account.MustParseAccount()

	contract := common.MustCreateContract(abiJSON)
	defer contract.Client.Close()

	// ensure owner privilege
	owner := common.MustCallAddress(contract, "owner")
	if owner != from {
		fmt.Println("Owner privilege required:", owner)
		os.Exit(1)
	}

	option := types.ContractMethodSendOption{
		From:     types.NewAddress(from),
		GasPrice: types.NewBigIntByRaw(account.MustParsePrice()),
	}

	// ensure contract is paused
	var paused bool
	common.MustCall(contract, &paused, "paused")
	if !paused {
		fmt.Print("Pause contract...")
		txHash := common.MustExecuteTx(contract, &option, "pause")
		fmt.Println("Done")
		fmt.Println("tx hash:", txHash)
	}

	// withdraw
	if len(recipient) == 0 {
		recipient = from
	}
	amount := account.MustParseValue()
	fmt.Print("Withdraw funds...")
	txHash := common.MustExecuteTx(contract, &option, "withdraw", common.MustAddress2Bytes20(recipient), amount)
	fmt.Println("Done")
	fmt.Println("tx hash:", txHash)
}
