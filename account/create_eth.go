package account

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "create-eth",
		Short: "Create a new address that compatible with Ethereum",
		Run:   createETHAccount,
	})
}

func createETHAccount(cmd *cobra.Command, args []string) {
	password := mustInputAndConfirmPassword()
	account, err := am.CreateEthCompatible(password)
	if err != nil {
		fmt.Println("Failed to create account:", err.Error())
		return
	}

	fmt.Println("New account:", account.String())
}
