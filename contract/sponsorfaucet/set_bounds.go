package sponsorfaucet

import (
	"fmt"
	"math/big"
	"os"

	"github.com/Conflux-Chain/conflux-toolkit/account"
	"github.com/Conflux-Chain/conflux-toolkit/contract/common"
	"github.com/Conflux-Chain/conflux-toolkit/util"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/spf13/cobra"
)

var (
	gasTotalCfx        string
	collateralTotalCfx string
	gasBoundCfx        string
	collateralBoundCfx string
	feeBoundGdrip      string
)

func init() {
	setBoundsCmd := &cobra.Command{
		Use:   "set-bounds",
		Short: "Set bounds for sponsor faucet (owner only)",
		Run:   setBounds,
	}

	account.AddAccountVar(setBoundsCmd)
	account.AddGasPriceVar(setBoundsCmd)

	setBoundsCmd.PersistentFlags().StringVar(&gasTotalCfx, "gas-total-cfx", "", "Gas to sponsor in total (CFX)")
	setBoundsCmd.MarkPersistentFlagRequired("gas-total-cfx")

	setBoundsCmd.PersistentFlags().StringVar(&collateralTotalCfx, "collateral-total-cfx", "", "Collateral to sponsor in total (CFX)")
	setBoundsCmd.MarkPersistentFlagRequired("collateral-total-cfx")

	setBoundsCmd.PersistentFlags().StringVar(&gasBoundCfx, "gas-bound-cfx", "", "Gas bound to sponsor at a time (CFX)")
	setBoundsCmd.MarkPersistentFlagRequired("gas-bound-cfx")

	setBoundsCmd.PersistentFlags().StringVar(&collateralBoundCfx, "collateral-bound-cfx", "", "Collateral bound to sponsor at a time (CFX)")
	setBoundsCmd.MarkPersistentFlagRequired("collateral-bound-cfx")

	setBoundsCmd.PersistentFlags().StringVar(&feeBoundGdrip, "fee-bound-gdrip", "", "Gas fee upper bound to sponsor at a time (Gdrip)")
	setBoundsCmd.MarkPersistentFlagRequired("fee-bound-gdrip")

	rootCmd.AddCommand(setBoundsCmd)
}

func setBounds(cmd *cobra.Command, args []string) {
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

	password := account.MustInputPassword("Enter password: ")
	account.DefaultAccountManager.Unlock(types.Address(from), password)

	gasTotal := util.MustParseBigInt(gasTotalCfx, 18)
	collateralTotal := util.MustParseBigInt(collateralTotalCfx, 18)
	gasBound := util.MustParseBigInt(gasBoundCfx, 18)
	collateralBound := util.MustParseBigInt(collateralBoundCfx, 18)
	feeBound := util.MustParseBigInt(feeBoundGdrip, 9)

	if new(big.Int).Mul(feeBound, big.NewInt(1000)).Cmp(gasBound) > 0 {
		fmt.Println("gas bound should >= gas fee upper bound * 1000")
		os.Exit(1)
	}

	fmt.Print(option)
	fmt.Println("Set bounds...")
	txHash := common.MustExecuteTx(contract, &option, "setBounds", gasTotal, collateralTotal, gasBound, collateralBound, feeBound)
	fmt.Println("tx hash:", txHash)
}
