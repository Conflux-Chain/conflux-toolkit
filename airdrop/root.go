package airdrop

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Conflux-Chain/conflux-toolkit/account"
	"github.com/Conflux-Chain/conflux-toolkit/rpc"
	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	clientRpc "github.com/Conflux-Chain/go-conflux-sdk/rpc"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "airdrop",
		Short: "airdrop subcommand",
		Run:   doAirdrop,
	}
	// path for record result
	resultPath = "./airdrop_result"
)

func init() {
	AddAirdropListFileVar(rootCmd)
	AddAirdropNumberVar(rootCmd)
	AddFromVar(rootCmd)
	AddBatchVar(rootCmd)
}

// SetParent sets parent command
func SetParent(parent *cobra.Command) {
	parent.AddCommand(rootCmd)
}

func doAirdrop(cmd *cobra.Command, args []string) {

	airdropInfos := mustParseInput()
	client, am, lastPoint, nonce := initialEnviorment()
	checkBalance(client, airdropInfos)

	count := 0
	rpcBatchElems := []clientRpc.BatchElem{}
	tx := &types.UnsignedTransaction{}
	var e error

	for i, v := range airdropInfos {
		// composite tx
		if i == 0 {
			tx, e = client.CreateUnsignedTransaction(types.Address(from), v.To, types.NewBigInt(0), nil)
			OsExitIfErr(e, "create unsigned tx error")
		}
		if i <= lastPoint {
			continue
		}

		tx.To = &v.To
		rawValue := big.NewInt(1).Mul(big.NewInt(int64(airdropNumber*v.Weight)), big.NewInt(1e18))
		tx.Value = types.NewBigIntByRaw(rawValue)
		tx.Nonce = types.NewBigIntByRaw(nonce)
		tx.GasPrice = types.NewBigInt(1)
		tx.Gas = types.NewBigInt(21000)

		// sign
		encoded, e := am.SignTransaction(*tx)
		OsExitIfErr(e, "Failed to sign transaction")
		fmt.Printf("sign to %v with value %v CFX done\n", tx.To, airdropNumber*v.Weight)

		// push to batch item array
		batchElemResult := types.Hash("")
		rpcBatchElems = append(rpcBatchElems, clientRpc.BatchElem{
			Method: "cfx_sendRawTransaction",
			Args:   []interface{}{hexutil.Encode(encoded)},
			Result: &batchElemResult,
		})

		count++
		if count == perBatchNum || i == len(airdropInfos)-1 {
			// batch send
			e := client.BatchCallRPC(rpcBatchElems)
			OsExitIfErr(e, "batch send error")

			// save record
			ioutil.WriteFile(resultPath, []byte(strconv.Itoa(i)), 0777)

			// wait last packed
			lastHash := rpcBatchElems[len(rpcBatchElems)-1].Result.(*types.Hash)

			fmt.Printf("send %v tx done, wait be executed:%v\n", len(rpcBatchElems), lastHash)
			_, e = client.WaitForTransationReceipt(*lastHash, time.Second)
			OsExitIfErr(e, "failed to get result of tx %+v", tx)

			// reset count and batch elem result
			rpcBatchElems = []clientRpc.BatchElem{}
			count = 0
		}

		if i == len(airdropInfos)-1 {
			e := os.Remove(resultPath)
			OsExitIfErr(e, "remove result file error.")
		}

		nonce = nonce.Add(nonce, big.NewInt(1))
	}
	fmt.Printf("airdrop done\n")
}

func mustParseInput() []AirdropInfo {
	// read csv file
	content, err := ioutil.ReadFile(airdropListFile)
	OsExitIfErr(err, "read file %v error", airdropListFile)

	// parse to struct
	lines := strings.Split(string(content), "\n")
	airdropInfos := []AirdropInfo{}
	for _, v := range lines {
		v = strings.Replace(v, "\t", " ", -1)
		v = strings.Replace(v, ",", " ", -1)
		v = strings.Join(strings.Fields(v), " ")
		items := strings.Split(v, " ")

		if len(items) != 2 {
			OsExit("elems length of %#v is %v not equal to 2\n", v, len(items))
		}

		weight, err := strconv.Atoi(items[1])
		OsExitIfErr(err, "parse %v to int error", weight)

		info := AirdropInfo{
			To:     types.Address(items[0]),
			Weight: weight,
		}
		airdropInfos = append(airdropInfos, info)
	}
	return airdropInfos
}

func initialEnviorment() (client *sdk.Client, am *sdk.AccountManager, lastPoint int, nonce *big.Int) {

	// foreach and send transaction
	am = account.DefaultAccountManager
	client = rpc.MustCreateClientWithRetry(100)
	client.SetAccountManager(am)

	password := account.MustInputPassword("Enter password: ")
	am.Unlock(types.Address(from), password)

	// get inital Nonce
	nonce, e := client.GetNextNonce(types.Address(from))
	OsExitIfErr(e, "get nonce of from %v", from)

	resultFs, e := os.OpenFile(resultPath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0777)
	OsExitIfErr(e, "failed to create file")
	defer resultFs.Close()

	lastPointStr, e := ioutil.ReadFile(resultPath)
	OsExitIfErr(e, "read result content error")

	if len(lastPointStr) > 0 {
		lastPoint, e = strconv.Atoi(string(lastPointStr))
	} else {
		lastPoint = 0
	}
	return
}

func checkBalance(client *sdk.Client, airdrops []AirdropInfo) {
	balance, err := client.GetBalance(types.Address(from))
	OsExitIfErr(err, "failed to get balance")

	var needCfx int64
	for _, v := range airdrops {
		needCfx = needCfx + int64(v.Weight*airdropNumber)
	}
	needDrip := big.NewInt(1).Mul(big.NewInt(needCfx), big.NewInt(1e18))

	if balance.Cmp(needDrip) < 0 {
		OsExit("out of balance, need %v, has %v:", needDrip, balance)
	}
}

// OsExitIfErr ...
func OsExitIfErr(err error, format string, a ...interface{}) {
	if err != nil {
		fmt.Printf(format, a...)
		fmt.Printf("--- error:%v", err)
		fmt.Println()
		os.Exit(1)
	}
}

// OsExit ...
func OsExit(format string, a ...interface{}) {
	fmt.Printf(format, a...)
	fmt.Println()
	os.Exit(1)
}
