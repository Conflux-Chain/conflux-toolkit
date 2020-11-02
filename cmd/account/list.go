package account

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List accounts in ASC order",
	Run: func(cmd *cobra.Command, args []string) {
		accounts := listAccountsAsc()
		if len(accounts) == 0 {
			fmt.Println("No account found!")
			return
		}

		for i, addr := range accounts {
			fmt.Printf("[%v]\t%v\n", i, addr)
		}

		fmt.Printf("Totally %v accounts found.\n", len(accounts))
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func listAccountsAsc() []string {
	var accounts []string

	for _, addr := range am.List() {
		accounts = append(accounts, addr.String())
	}

	sort.Strings(accounts)

	return accounts
}
