package account

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	numAccounts uint

	createCmd = &cobra.Command{
		Use:   "create",
		Short: "Create new accounts",
		Run: func(cmd *cobra.Command, args []string) {
			createNewAccounts()
		},
	}
)

func init() {
	createCmd.PersistentFlags().UintVar(&numAccounts, "num", 1, "Number of accounts to create")

	rootCmd.AddCommand(createCmd)
}

func createNewAccounts() {
	password := mustInputAndConfirmPassword()

	for i := uint(0); i < numAccounts; i++ {
		account, err := am.Create(password)
		if err != nil {
			fmt.Println("Failed to create account:", err.Error())
			return
		}

		fmt.Println(account.String())
	}

	fmt.Printf("Totally %v new accounts created.\n", numAccounts)
}
