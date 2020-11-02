package account

import (
	"fmt"

	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an account",
	Run: func(cmd *cobra.Command, args []string) {
		deleteAccount()
	},
}

func init() {
	deleteCmd.PersistentFlags().StringVar(&account, "account", "", "Account address in HEX format or address index number")
	deleteCmd.MarkPersistentFlagRequired("account")

	rootCmd.AddCommand(deleteCmd)
}

func deleteAccount() {
	account := mustParseAccount()
	password := mustInputPassword("Enter password: ")

	if err := am.Delete(types.Address(account), password); err != nil {
		fmt.Println("Failed to delete account:", err.Error())
		return
	}

	fmt.Println("Account deleted!")
}
