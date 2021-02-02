package common

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Conflux-Chain/conflux-toolkit/account"
	"github.com/Conflux-Chain/conflux-toolkit/contract/abi"
	"github.com/Conflux-Chain/conflux-toolkit/rpc"
	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/spf13/cobra"
)

var contract string

// AddContractVar adds contract variable for specified command.
func AddContractVar(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&contract, "contract", "", "Contract address in HEX format")
	cmd.MarkPersistentFlagRequired("contract")
}

// MustCreateContract creates an instance to interact with contract.
func MustCreateContract(abiJSON string) *sdk.Contract {
	return MustGetContract(abiJSON, contract)
}

// MustGetContract creates an instance to interact with contract at contractAddress.
func MustGetContract(abiJSON string, contractAddress string) *sdk.Contract {
	client := rpc.MustCreateClient()
	client.SetAccountManager(account.DefaultAccountManager)

	contract, err := client.GetContract([]byte(abiJSON), types.NewAddress(contractAddress))
	if err != nil {
		fmt.Println("Failed to create contract instance:", err.Error())
		os.Exit(1)
	}

	return contract
}

func MustGetCTokenContract(contractAddress string) *sdk.Contract {
	return MustGetContract(abi.GetCTokenABI(), contractAddress)
}

// MustCall call contract for the specified method and arguments.
func MustCall(contract *sdk.Contract, resultPtr interface{}, method string, args ...interface{}) {
	if err := contract.Call(nil, resultPtr, method, args...); err != nil {
		fmt.Printf("Failed to call method %v: %v\n", method, err.Error())
		os.Exit(1)
	}
}

// MustCallAddress call contract and return address value for the specified method and arguments.
func MustCallAddress(contract *sdk.Contract, method string, args ...interface{}) string {
	var result [20]byte
	MustCall(contract, &result, method, args...)
	return types.NewBytes(result[:]).String()
}

// MustExecuteTx call contract and wait for the succeeded receipt.
func MustExecuteTx(contract *sdk.Contract, option *types.ContractMethodSendOption, method string, args ...interface{}) string {
	txHash, err := contract.SendTransaction(option, method, args...)
	if err != nil {
		fmt.Println("Failed to send transaction:", err.Error())
		os.Exit(1)
	}

	for {
		time.Sleep(time.Second)

		receipt, err := contract.Client.GetTransactionReceipt(*txHash)
		if err != nil {
			fmt.Println("Failed to get receipt:", err.Error())
			os.Exit(1)
		}

		if receipt == nil {
			continue
		}

		if receipt.OutcomeStatus == 0 {
			break
		}

		fmt.Println("Receipt outcome status is:", receipt.OutcomeStatus)
		os.Exit(1)
	}

	return txHash.String()
}

// MustAddress2Bytes20 converts address in HEX format to [20]byte.
func MustAddress2Bytes20(address string) [20]byte {
	if strings.HasPrefix(address, "0x") {
		address = address[2:]
	}

	decoded, err := hex.DecodeString(address)
	if err != nil {
		fmt.Println("Failed to decode address to [20]byte:", err.Error())
		os.Exit(1)
	}

	if len(decoded) != 20 {
		fmt.Println("Failed to decode address to [20]byte, invalid length", len(decoded))
		os.Exit(1)
	}

	var result [20]byte
	copy(result[:], decoded)

	return result
}
