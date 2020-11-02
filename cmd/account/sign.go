package account

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"os"

	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/Conflux-Chain/go-conflux-sdk/utils"
	"github.com/spf13/cobra"
)

var (
	nonce        uint32
	to           string
	priceStr     string
	gasLimit     uint32
	valueStr     string
	storageLimit uint64
	epoch        uint64
	chain        uint
	data         string

	dripsPerCfx *big.Float = new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))

	signCmd = &cobra.Command{
		Use:   "sign",
		Short: "Sign transaction to send",
		Run: func(cmd *cobra.Command, args []string) {
			sign()
		},
	}
)

func init() {
	signCmd.PersistentFlags().StringVar(&account, "from", "0", "From address in HEX format or address index number")
	signCmd.MarkPersistentFlagRequired("from")

	signCmd.PersistentFlags().Uint32Var(&nonce, "nonce", 0, "Transaction nonce")
	signCmd.MarkPersistentFlagRequired("nonce")

	signCmd.PersistentFlags().StringVar(&to, "to", "", "To address in HEX format")
	signCmd.MarkPersistentFlagRequired("to")

	signCmd.PersistentFlags().StringVar(&priceStr, "price", "1", "Gas price in drip")

	signCmd.PersistentFlags().Uint32Var(&gasLimit, "gas", 21000, "Gas limit")

	signCmd.PersistentFlags().StringVar(&valueStr, "value", "", "Value to transfer in CFX")
	signCmd.MarkPersistentFlagRequired("value")

	signCmd.PersistentFlags().Uint64Var(&storageLimit, "storage", 0, "Storage limit")

	signCmd.PersistentFlags().Uint64Var(&epoch, "epoch", 0, "Transaction epoch height")
	signCmd.MarkPersistentFlagRequired("epoch")

	signCmd.PersistentFlags().UintVar(&chain, "chain", 1029, "Conflux chain ID")

	signCmd.PersistentFlags().StringVar(&data, "data", "", "Transaction data or encoded ABI data in HEX format")

	rootCmd.AddCommand(signCmd)
}

func sign() {
	tx := types.UnsignedTransaction{
		UnsignedTransactionBase: types.UnsignedTransactionBase{
			From:         types.NewAddress(mustParseAccount()),
			Nonce:        types.NewBigInt(int64(nonce)),
			GasPrice:     types.NewBigIntByRaw(mustParsePrice()),
			Gas:          types.NewBigInt(int64(gasLimit)),
			Value:        types.NewBigIntByRaw(mustParseValue()),
			StorageLimit: types.NewUint64(storageLimit),
			EpochHeight:  types.NewUint64(epoch),
			ChainID:      types.NewUint(chain),
		},
		To: types.NewAddress(to),
	}

	if len(data) > 0 {
		txData, err := utils.HexStringToBytes(data)
		if err != nil {
			fmt.Println("Invalid tx data:", err.Error())
			return
		}
		tx.Data = txData
	}

	password := mustInputPassword("Enter password: ")

	encoded, err := am.SignAndEcodeTransactionWithPassphrase(tx, password)
	if err != nil {
		fmt.Println("Failed to sign transaction:", err.Error())
		return
	}

	fmt.Println("=======================================")
	fmt.Println("0x" + hex.EncodeToString(encoded))
}

func mustParsePrice() *big.Int {
	price, ok := new(big.Int).SetString(priceStr, 10)
	if !ok {
		fmt.Println("invalid number format for price")
		os.Exit(1)
	}

	return price
}

func mustParseValue() *big.Int {
	value, ok := new(big.Float).SetString(valueStr)
	if !ok {
		fmt.Println("invalid float format for value")
		os.Exit(1)
	}

	result, _ := new(big.Float).Mul(value, dripsPerCfx).Int(nil)

	return result
}
