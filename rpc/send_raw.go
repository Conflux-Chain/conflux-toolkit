package rpc

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	sendCmd := &cobra.Command{
		Use:   "send-raw",
		Short: "Send signed transaction",
		Run:   sendRaw,
	}

	sendCmd.PersistentFlags().StringVar(&data, "raw", "", "Raw transaction in HEX format")
	sendCmd.MarkPersistentFlagRequired("raw")

	rootCmd.AddCommand(sendCmd)
}

func sendRaw(cmd *cobra.Command, args []string) {
	client := MustCreateClient()
	defer client.Close()

	if strings.HasPrefix(data, "0x") {
		data = data[2:]
	}

	rawData, err := hex.DecodeString(data)
	if err != nil {
		fmt.Println("Failed to decode raw data in HEX format:", err.Error())
		return
	}

	txHash, err := client.SendRawTransaction(rawData)
	if err != nil {
		fmt.Println("Failed to send raw transaction:", err.Error())
		return
	}

	fmt.Println("Transaction sent:", txHash)
}
