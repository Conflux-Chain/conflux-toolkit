package util

import (
	"fmt"
	"os"
	"time"

	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
)

// URL is the CFX URL to create connection.
var URL string

// Contract is contract address in HEX format.
var Contract string

// MustGetClient creates an connection to full node.
func MustGetClient() *sdk.Client {
	client, err := sdk.NewClientWithRetry(URL, 3, time.Second)
	if err != nil {
		fmt.Println("Failed to create client:", err.Error())
		os.Exit(1)
	}

	return client
}

// MustCall calls contract with specified ABI encoded data.
func MustCall(data string) string {
	client := MustGetClient()
	defer client.Close()

	request := types.CallRequest{
		To:   types.NewAddress(Contract),
		Data: data,
	}

	encoded, err := client.Call(request, nil)
	if err != nil {
		fmt.Println("Failed to call:", err.Error())
		os.Exit(1)
	}

	return *encoded
}
