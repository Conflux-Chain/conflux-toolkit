package account

import (
	"fmt"

	"github.com/spf13/cobra"
)

var privateKey string

func init() {
	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Import private key",
		Run:   importKey,
	}

	importCmd.PersistentFlags().StringVar(&privateKey, "key", "", "Private key in HEX format")
	importCmd.MarkPersistentFlagRequired("key")

	rootCmd.AddCommand(importCmd)
}

func importKey(cmd *cobra.Command, args []string) {
	password := mustInputAndConfirmPassword()

	account, err := am.ImportKey(privateKey, password)
	if err != nil {
		fmt.Println("Failed to import private key:", err.Error())
		return
	}

	fmt.Println("Imported account:", account.String())
}
