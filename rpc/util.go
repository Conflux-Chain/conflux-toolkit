package rpc

import (
	"time"

	"github.com/Conflux-Chain/conflux-toolkit/util"
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
	return MustCreateClientWithRetry(3)
}

// MustCreateClientWithRetry creates an connection to full node.
func MustCreateClientWithRetry(retryCount int) *sdk.Client {
	client, err := sdk.NewClient(url, sdk.ClientOption{
		RetryCount:    retryCount,
		RetryInterval: time.Second})
	if err != nil {
		util.OsExitIfErr(err, "Failed to create client")
		// fmt.Println("Failed to create client:", err.Error())
		// os.Exit(1)
	}

	return client
}
