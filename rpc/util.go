package rpc

import (
	"sync"
	"time"

	"github.com/Conflux-Chain/conflux-toolkit/util"
	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	"github.com/spf13/cobra"
)

var (
	url         string
	clientCache map[string]*sdk.Client = make(map[string]*sdk.Client)
	clientMutex sync.Mutex
)

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
	if clientCache[url] == nil {
		clientMutex.Lock()
		defer clientMutex.Unlock()
		if clientCache[url] == nil {
			var err error
			clientCache[url], err = sdk.NewClient(url, sdk.ClientOption{
				RetryCount:    retryCount,
				RetryInterval: time.Second})
			if err != nil {
				util.OsExitIfErr(err, "Failed to create client")
			}
		}
	}
	return clientCache[url]
}
