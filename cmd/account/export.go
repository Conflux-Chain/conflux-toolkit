package account

import (
	"fmt"
	"os"
	"strconv"

	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/spf13/cobra"
)

var (
	account string

	exportCmd = &cobra.Command{
		Use:   "export",
		Short: "Export private key",
		Run: func(cmd *cobra.Command, args []string) {
			exportKey()
		},
	}
)

func init() {
	exportCmd.PersistentFlags().StringVar(&account, "account", "", "Account address in HEX format or address index number")
	exportCmd.MarkPersistentFlagRequired("account")

	rootCmd.AddCommand(exportCmd)
}

func exportKey() {
	account := mustParseAccount()
	password := mustInputPassword("Enter password: ")

	privKey, err := am.Export(types.Address(account), password)
	if err != nil {
		fmt.Println("Failed to export private key:", err.Error())
		return
	}

	fmt.Println("Private key:", privKey)
}

func mustParseAccount() string {
	accountIndex, err := strconv.Atoi(account)
	if err != nil {
		return account
	}

	accounts := listAccountsAsc()
	if len(accounts) == 0 {
		fmt.Println("No account found!")
		os.Exit(1)
	}

	if accountIndex >= len(accounts) {
		fmt.Println("Invalid account index, it should be between 0 and", len(accounts)-1)
		os.Exit(1)
	}

	fmt.Println("Account:", accounts[accountIndex])

	return accounts[accountIndex]
}
