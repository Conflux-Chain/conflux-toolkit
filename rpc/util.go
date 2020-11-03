package rpc

import (
	"fmt"
	"os"
	"time"

	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	"github.com/spf13/cobra"
)

var url string

// AddURLVar adds URL variable for specified command
func AddURLVar(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&url, "url", "http://main.confluxrpc.org", "Conflux RPC URL")
}

// MustCreateClient creates an connection to full node.
func MustCreateClient() *sdk.Client {
	client, err := sdk.NewClientWithRetry(url, 3, time.Second)
	if err != nil {
		fmt.Println("Failed to create client:", err.Error())
		os.Exit(1)
	}

	return client
}
