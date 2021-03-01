package account

import (
	"fmt"

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
	password := MustInputPassword("Enter password: ")

	if err := am.Delete(*account, password); err != nil {
		fmt.Println("Failed to delete account:", err.Error())
		return
	}

	fmt.Println("Account deleted!")
}
