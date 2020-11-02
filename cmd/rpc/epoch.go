package rpc

import (
	"fmt"
	"math/big"
	"os"

	"github.com/Conflux-Chain/conflux-toolkit/util"
	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/spf13/cobra"
)

var epochCmd = &cobra.Command{
	Use:   "epoch",
	Short: "Get epoch info",
	Run: func(cmd *cobra.Command, args []string) {
		getEpochInfo()
	},
}

func init() {
	rootCmd.AddCommand(epochCmd)
}

func getEpochInfo() {
	client := util.MustGetClient()

	mined := mustGetEpochNumber(client, types.EpochLatestMined)
	fmt.Printf("Latest mined      : %v\n", mined)

	stated := mustGetEpochNumber(client, types.EpochLatestState)
	fmt.Printf("Latest state      : %v (%v)\n", stated, new(big.Int).Sub(mined, stated))

	confirmed := mustGetEpochNumber(client, types.EpochLatestConfirmed)
	fmt.Printf("Latest confirmed  : %v (%v)\n", confirmed, new(big.Int).Sub(mined, confirmed))

	checkpoint := mustGetEpochNumber(client, types.EpochLatestCheckpoint)
	fmt.Printf("Latest checkpoint : %v (%v)\n", checkpoint, new(big.Int).Sub(mined, checkpoint))
}

func mustGetEpochNumber(client *sdk.Client, epoch *types.Epoch) *big.Int {
	num, err := client.GetEpochNumber(epoch)
	if err != nil {
		fmt.Println("Failed to get epoch number:", err.Error())
		os.Exit(1)
	}

	return num
}
