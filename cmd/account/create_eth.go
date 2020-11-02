package account

import (
	"fmt"

	"github.com/spf13/cobra"
)

var createETHCmd = &cobra.Command{
	Use:   "create-eth",
	Short: "Create a new address that compatible with Ethereum",
	Run: func(cmd *cobra.Command, args []string) {
		createETHAccount()
	},
}

func init() {
	rootCmd.AddCommand(createETHCmd)
}

func createETHAccount() {
	password := mustInputAndConfirmPassword()

	account, err := am.CreateEthCompatible(password)
	if err != nil {
		fmt.Println("Failed to create account:", err.Error())
		return
	}

	fmt.Println("New account:", account.String())
}
