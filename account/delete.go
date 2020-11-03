package account

import (
	"fmt"

	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/spf13/cobra"
)

func init() {
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an account",
		Run:   deleteAccount,
	}

	AddAccountVar(deleteCmd)

	rootCmd.AddCommand(deleteCmd)
}

func deleteAccount(cmd *cobra.Command, args []string) {
	account := MustParseAccount()
	password := mustInputPassword("Enter password: ")

	if err := am.Delete(types.Address(account), password); err != nil {
		fmt.Println("Failed to delete account:", err.Error())
		return
	}

	fmt.Println("Account deleted!")
}
