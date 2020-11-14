package transfer

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Conflux-Chain/conflux-toolkit/account"
	"github.com/Conflux-Chain/conflux-toolkit/rpc"
	"github.com/Conflux-Chain/conflux-toolkit/util"
	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	clientRpc "github.com/Conflux-Chain/go-conflux-sdk/rpc"
	"github.com/Conflux-Chain/go-conflux-sdk/types"

	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "transfer",
		Short: "transfer subcommand",
		Run:   doTransfers,
	}
	// path for record result
	resultPath      = "./transfer_result"
	defaultGasLimit = types.NewBigInt(21000)

	// command flags
	receiverListFile string
	receiveNumber    uint
	// from            string
	perBatchNum uint
)

func init() {
	rpc.AddURLVar(rootCmd)
	account.AddFromVar(rootCmd)
	account.AddGasPriceVar(rootCmd)

	rootCmd.PersistentFlags().StringVar(&receiverListFile, "receivers", "", "receiver list file path")
	rootCmd.MarkPersistentFlagRequired("receivers")

	rootCmd.PersistentFlags().UintVar(&receiveNumber, "number", 1, "send value in CFX")
	rootCmd.MarkPersistentFlagRequired("number")

	rootCmd.PersistentFlags().UintVar(&perBatchNum, "batch", 100, "send tx number per batch")
}

// SetParent sets parent command
func SetParent(parent *cobra.Command) {
	parent.AddCommand(rootCmd)
}

func doTransfers(cmd *cobra.Command, args []string) {

	receiverInfos := mustParseInput()
	client, am, from, lastPoint, nonce := initialEnviorment()
	checkBalance(client, from, receiverInfos)

	count := uint(0)
	rpcBatchElems := []clientRpc.BatchElem{}
	tx := &types.UnsignedTransaction{}
	var e error

	for i, v := range receiverInfos {
		// composite tx
		if i == 0 {
			tx, e = client.CreateUnsignedTransaction(from, v.Address, types.NewBigInt(0), nil)
			util.OsExitIfErr(e, "create unsigned tx error")
		}
		if i <= lastPoint {
			continue
		}

		tx.To = &v.Address
		rawValue := big.NewInt(1).Mul(big.NewInt(int64(receiveNumber*v.Weight)), big.NewInt(1e18))
		tx.Value = types.NewBigIntByRaw(rawValue)
		tx.Nonce = types.NewBigIntByRaw(nonce)
		tx.GasPrice = types.NewBigIntByRaw(account.MustParsePrice())
		tx.Gas = defaultGasLimit

		// sign
		encoded, e := am.SignTransaction(*tx)
		util.OsExitIfErr(e, "Failed to sign transaction")
		fmt.Printf("sign to %v with value %v CFX done\n", tx.To, receiveNumber*v.Weight)

		// push to batch item array
		batchElemResult := types.Hash("")
		rpcBatchElems = append(rpcBatchElems, clientRpc.BatchElem{
			Method: "cfx_sendRawTransaction",
			Args:   []interface{}{"0x" + hex.EncodeToString(encoded)},
			Result: &batchElemResult,
		})

		count++
		if count == perBatchNum || i == len(receiverInfos)-1 {
			// batch send
			e := client.BatchCallRPC(rpcBatchElems)
			util.OsExitIfErr(e, "batch send error")

			// save record
			ioutil.WriteFile(resultPath, []byte(strconv.Itoa(i)), 0777)

			// wait last packed
			lastHash := rpcBatchElems[len(rpcBatchElems)-1].Result.(*types.Hash)

			fmt.Printf("batch send %v tx, total send %v done, wait last be executed: %v\n", len(rpcBatchElems), i+1, lastHash)
			_, e = client.WaitForTransationReceipt(*lastHash, time.Second)
			util.OsExitIfErr(e, "failed to get result of tx %+v", tx)

			// reset count and batch elem result
			rpcBatchElems = []clientRpc.BatchElem{}
			count = 0
		}

		if i == len(receiverInfos)-1 {
			e := os.Remove(resultPath)
			util.OsExitIfErr(e, "remove result file error.")
		}

		nonce = nonce.Add(nonce, big.NewInt(1))
	}
	fmt.Printf("transfer done\n")
}

func mustParseInput() []Receiver {
	// read csv file
	content, err := ioutil.ReadFile(receiverListFile)
	util.OsExitIfErr(err, "read file %v error", receiverListFile)

	// parse to struct
	lines := strings.Split(string(content), "\n")
	receiverInfos := []Receiver{}
	for _, v := range lines {
		v = strings.Replace(v, "\t", " ", -1)
		v = strings.Replace(v, ",", " ", -1)
		items := strings.Fields(v)
		if len(v) == 0 {
			continue
		}

		if len(items) != 2 {
			util.OsExit("elems length of %#v is %v not equal to 2\n", v, len(items))
		}

		weight, err := strconv.Atoi(items[1])
		util.OsExitIfErr(err, "parse %v to int error", weight)

		info := Receiver{
			Address: types.Address(items[0]),
			Weight:  uint(weight),
		}
		receiverInfos = append(receiverInfos, info)
	}
	fmt.Printf("receiver list count :%+v\n", len(receiverInfos))
	return receiverInfos
}

func initialEnviorment() (client *sdk.Client, am *sdk.AccountManager, from types.Address, lastPoint int, nonce *big.Int) {

	am = account.DefaultAccountManager
	client = rpc.MustCreateClientWithRetry(100)
	client.SetAccountManager(am)

	from = types.Address(account.MustParseAccount())

	password := account.MustInputPassword("Enter password: ")
	am.Unlock(from, password)

	// get inital Nonce
	nonce, e := client.GetNextNonce(from)
	util.OsExitIfErr(e, "get nonce of from %v", from)

	resultFs, e := os.OpenFile(resultPath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0777)
	util.OsExitIfErr(e, "failed to create file")
	defer resultFs.Close()

	lastPointStr, e := ioutil.ReadFile(resultPath)
	util.OsExitIfErr(e, "read result content error")

	if len(lastPointStr) > 0 {
		lastPoint, e = strconv.Atoi(string(lastPointStr))
	} else {
		lastPoint = -1
	}
	return
}

func checkBalance(client *sdk.Client, from types.Address, receivers []Receiver) {
	balance, err := client.GetBalance(from)
	util.OsExitIfErr(err, "failed to get balance")

	need := big.NewInt(0)
	for _, v := range receivers {
		receiverNeed := big.NewInt(1).Mul(big.NewInt(int64(v.Weight*receiveNumber)), big.NewInt(1e18))
		gasFee := big.NewInt(1).Mul(defaultGasLimit.ToInt(), account.MustParsePrice())
		need = need.Add(need, receiverNeed)
		need = need.Add(need, gasFee)
	}

	if balance.Cmp(need) < 0 {
		lastPointStr, e := ioutil.ReadFile(resultPath)
		util.OsExitIfErr(e, "read result content error")

		if len(lastPointStr) == 0 {
			os.Remove(resultPath)
		}
		util.OsExit("out of balance, need %v, has %v", util.DisplayValueWithUnit(need), util.DisplayValueWithUnit(balance))
	}
	fmt.Printf("balance is enough, need %v, has %v\n", util.DisplayValueWithUnit(need), util.DisplayValueWithUnit(balance))
}
