package account

import (
	"fmt"

	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export private key",
	Run: func(cmd *cobra.Command, args []string) {
		exportKey()
	},
}

func init() {
	exportCmd.PersistentFlags().StringVar(&Account, "account", "", "Account address in HEX format or address index number")
	exportCmd.MarkPersistentFlagRequired("account")

	rootCmd.AddCommand(exportCmd)
}

func exportKey() {
	account := MustParseAccount()
	password := mustInputPassword("Enter password: ")

	privKey, err := am.Export(types.Address(account), password)
	if err != nil {
		fmt.Println("Failed to export private key:", err.Error())
		return
	}

	fmt.Println("Private key:", privKey)
}
