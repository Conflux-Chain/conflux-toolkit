package account

import (
	"fmt"

	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/spf13/cobra"
)

func init() {
	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export private key",
		Run:   exportKey,
	}

	AddAccountVar(exportCmd)

	rootCmd.AddCommand(exportCmd)
}

func exportKey(cmd *cobra.Command, args []string) {
	account := MustParseAccount()
	password := MustInputPassword("Enter password: ")

	privKey, err := am.Export(types.Address(account), password)
	if err != nil {
		fmt.Println("Failed to export private key:", err.Error())
		return
	}

	fmt.Println("Private key:", privKey)
}
