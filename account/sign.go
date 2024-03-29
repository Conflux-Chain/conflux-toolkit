package account

import (
	"encoding/hex"
	"fmt"

	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/Conflux-Chain/go-conflux-sdk/types/cfxaddress"
	"github.com/Conflux-Chain/go-conflux-sdk/utils"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/spf13/cobra"
)

var (
	nonce        uint32
	to           string
	gasLimit     uint32
	storageLimit uint64
	epoch        uint64
	chain        uint
	data         string
	space        string
)

func init() {
	signCmd := &cobra.Command{
		Use:   "sign",
		Short: "Sign transaction to send",
		Run:   sign,
	}

	AddFromVar(signCmd)

	signCmd.PersistentFlags().StringVar(&space, "space", "c", "'c' represents core space and 'e' represents espace")

	signCmd.PersistentFlags().Uint32Var(&nonce, "nonce", 0, "Transaction nonce")
	signCmd.MarkPersistentFlagRequired("nonce")

	signCmd.PersistentFlags().StringVar(&to, "to", "", "To address in HEX format")
	signCmd.MarkPersistentFlagRequired("to")

	AddGasPriceVar(signCmd)

	signCmd.PersistentFlags().Uint32Var(&gasLimit, "gas", 21000, "Gas limit")

	signCmd.PersistentFlags().StringVar(&ValueCfx, "value", "", "Value to transfer in CFX")
	signCmd.MarkPersistentFlagRequired("value")

	signCmd.PersistentFlags().Uint64Var(&storageLimit, "storage", 0, "Storage limit")

	signCmd.PersistentFlags().Uint64Var(&epoch, "epoch", 0, "Transaction epoch height")
	signCmd.MarkPersistentFlagRequired("epoch")

	signCmd.PersistentFlags().UintVar(&chain, "chain", 1029, "Conflux chain ID")

	signCmd.PersistentFlags().StringVar(&data, "data", "", "Transaction data or encoded ABI data in HEX format")

	rootCmd.AddCommand(signCmd)
}

func sign(cmd *cobra.Command, args []string) {
	to := cfxaddress.MustNew(to)
	tx := types.UnsignedTransaction{
		UnsignedTransactionBase: types.UnsignedTransactionBase{
			From:         MustParseAccount(),
			Nonce:        types.NewBigInt(uint64(nonce)),
			GasPrice:     types.NewBigIntByRaw(MustParsePrice()),
			Gas:          types.NewBigInt(uint64(gasLimit)),
			Value:        types.NewBigIntByRaw(MustParseValue()),
			StorageLimit: types.NewUint64(storageLimit),
			EpochHeight:  types.NewUint64(epoch),
			ChainID:      types.NewUint(chain),
		},
		To: &to,
	}

	if len(data) > 0 {
		txData, err := utils.HexStringToBytes(data)
		if err != nil {
			fmt.Println("Invalid tx data:", err.Error())
			return
		}
		tx.Data = txData
	}

	password := MustInputPassword("Enter password: ")

	var encoded []byte
	switch space {
	case "e":
		ethTx, from, chainID := CfxToEthTx(&tx)
		signedTx := SignEthLegacyTxWithPasswd(ethKeystore, from, ethTx, chainID, password)
		v, err := rlp.EncodeToBytes(signedTx)
		if err != nil {
			fmt.Println("Failed to sign espace transaction:", err.Error())
			return
		}
		// var result interface{}
		// rlp.DecodeBytes(v, &result)
		// fmt.Printf("decoded: %v\n", result)
		encoded = v
	default:
		v, err := am.SignAndEcodeTransactionWithPassphrase(tx, password)
		if err != nil {
			fmt.Println("Failed to sign core space transaction:", err.Error())
			return
		}
		encoded = v
	}

	fmt.Println("=======================================")
	fmt.Println("0x" + hex.EncodeToString(encoded))
}
