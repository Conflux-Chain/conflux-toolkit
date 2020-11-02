package rpc

import (
	"fmt"

	"github.com/Conflux-Chain/conflux-toolkit/util"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/spf13/cobra"
)

var txCmd = &cobra.Command{
	Use:   "tx",
	Short: "Get transaction by hash",
	Run: func(cmd *cobra.Command, args []string) {
		getTransaction()
	},
}

func init() {
	txCmd.PersistentFlags().StringVar(&hash, "hash", "", "Transaction hash in HEX format")
	txCmd.MarkPersistentFlagRequired("hash")

	rootCmd.AddCommand(txCmd)
}

func getTransaction() {
	client := util.MustGetClient()

	tx, err := client.GetTransactionByHash(types.Hash(hash))
	if err != nil {
		fmt.Println("Failed to get transaction:", err.Error())
		return
	}

	if tx == nil {
		fmt.Println("Transaction not found.")
		return
	}

	prettyPrintTx(tx)
}

func prettyPrintTx(tx *types.Transaction) {
	m := linkedhashmap.New()

	m.Put("hash", tx.Hash)
	m.Put("from", tx.From)
	m.Put("to", tx.To)
	m.Put("nonce", tx.Nonce.ToInt())
	m.Put("value", util.DisplayValueWithUnit(tx.Value.ToInt()))
	m.Put("gasPrice", util.DisplayValueWithUnit(tx.GasPrice.ToInt()))
	m.Put("gasLimit", tx.Gas.ToInt())
	m.Put("storageLimit", tx.StorageLimit.ToInt())
	m.Put("epoch", tx.EpochHeight.ToInt())
	m.Put("chainId", tx.ChainID.ToInt())
	if tx.Status == nil {
		m.Put("status", nil)
	} else {
		m.Put("status", uint64(*tx.Status))
	}
	m.Put("contractCreated", tx.ContractCreated)
	m.Put("blockHash", tx.BlockHash)
	if tx.TransactionIndex == nil {
		m.Put("transactionIndex", nil)
	} else {
		m.Put("transactionIndex", uint64(*tx.TransactionIndex))
	}
	m.Put("v", tx.V.ToInt())
	m.Put("r", tx.R)
	m.Put("s", tx.S)
	m.Put("data", tx.Data)

	content, err := m.ToJSON()
	if err != nil {
		fmt.Println("Failed to marshal data to JSON:", err.Error())
	} else {
		fmt.Println(string(content))
	}
}
