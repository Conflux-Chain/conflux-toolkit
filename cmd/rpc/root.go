package rpc

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Conflux-Chain/conflux-toolkit/util"
	"github.com/spf13/cobra"
)

var (
	hash    string
	address string
	data    string

	rootCmd = util.CreateUsageCommand("rpc", "RPC subcommand")
)

// SetParent sets parent command
func SetParent(parent *cobra.Command) {
	parent.AddCommand(rootCmd)
}

func mustPrintPrettyJSON(v interface{}) {
	content, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		fmt.Println("Failed to encode data to JSON:", err.Error())
		os.Exit(1)
	}

	fmt.Println(string(content))
}
