package account

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List accounts in ASC order",
		Run:   listAccounts,
	})
}

func listAccounts(cmd *cobra.Command, args []string) {
	accounts, ethAccounts := listAccountsAsc()
	if len(accounts) == 0 {
		fmt.Println("No account found!")
		return
	}

	for i := 0; i < len(accounts); i++ {
		fmt.Printf("[%v]\t%v\teSpace: %v\n", i, accounts[i], ethAccounts[i])
	}

	fmt.Printf("Totally %v accounts found.\n", len(accounts))
}
