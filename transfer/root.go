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

	"github.com/shopspring/decimal"

	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "transfer",
		Short: "transfer subcommand",
		Run:   doTransfers,
	}
	// path for record result
	resultPath      = "./transfer_result.txt"
	warnPath        = "./transfer_warn.txt"
	defaultGasLimit = types.NewBigInt(21000)

	// command flags
	receiverListFile string
	receiveNumber    decimal.Decimal
	// from            string
	perBatchNum uint
)

func init() {
	rpc.AddURLVar(rootCmd)
	account.AddFromVar(rootCmd)
	account.AddGasPriceVar(rootCmd)

	rootCmd.PersistentFlags().StringVar(&receiverListFile, "receivers", "", "receiver list file path")
	rootCmd.MarkPersistentFlagRequired("receivers")

	receiveNumberInStr := ""
	rootCmd.PersistentFlags().StringVar(&receiveNumberInStr, "number", "1", "send value in CFX")
	rootCmd.MarkPersistentFlagRequired("number")

	rootCmd.PersistentFlags().UintVar(&perBatchNum, "batch", 1000, "send tx number per batch")
	formatReceiverNumber(receiveNumberInStr)
}

func formatReceiverNumber(receiveNumberInStr string) {
	var err error
	receiveNumber, err = decimal.NewFromString(receiveNumberInStr)
	util.OsExitIfErr(err, "receiveNumber %v is not a number", receiveNumberInStr)
}

// SetParent sets parent command
func SetParent(parent *cobra.Command) {
	parent.AddCommand(rootCmd)
}

func doTransfers(cmd *cobra.Command, args []string) {

	resultFs, warnFs := creatRecordFiles()
	defer func() {
		resultFs.Close()
		warnFs.Close()
	}()

	receiverInfos := mustParseInput()
	client, am, lastPoint, from, nonce, chainID, epochHeight := initialEnviorment()
	checkBalance(client, from, receiverInfos)

	sendCount := uint(0)
	failCount := uint(0)
	rpcBatchElems := []clientRpc.BatchElem{}

	for i, v := range receiverInfos {
		if types.NormalAddress != v.Address.GetAddressType() {
			if failCount == 0 {
				warnFs.WriteString("=======invalid addresses==========\n")
			}
			msg := fmt.Sprintf("%v. *****Invalid address: %v", i+1, v.Address)
			fmt.Println(msg)
			_, err := warnFs.WriteString(msg)
			if err != nil {
				fmt.Printf("Fail to write warn msg \" %v \"\n", msg)
			}
			failCount++
			continue
		}

		if i <= lastPoint {
			continue
		}

		tx := createTx(from, v, nonce, chainID, epochHeight)

		// sign
		encoded, e := am.SignTransaction(*tx)
		util.OsExitIfErr(e, "Failed to sign transaction %+v", tx)
		fmt.Printf("%v. Sign to %v with value %v done\n", i+1, tx.To, util.DisplayValueWithUnit(tx.Value.ToInt()))

		// push to batch item array
		batchElemResult := types.Hash("")
		rpcBatchElems = append(rpcBatchElems, clientRpc.BatchElem{
			Method: "cfx_sendRawTransaction",
			Args:   []interface{}{"0x" + hex.EncodeToString(encoded)},
			Result: &batchElemResult,
		})

		sendCount++
		if sendCount == perBatchNum || i == len(receiverInfos)-1 {
			// batch send
			e := client.BatchCallRPC(rpcBatchElems)
			util.OsExitIfErr(e, "Batch send error")

			// save record
			ioutil.WriteFile(resultPath, []byte(strconv.Itoa(i)), 0777)

			// wait last packed
			lastHash := rpcBatchElems[len(rpcBatchElems)-1].Result.(*types.Hash)

			fmt.Printf("Batch send %v tx, total send %v done, failed %v, wait last be executed: %v\n", len(rpcBatchElems), uint(i)+1-failCount, failCount, lastHash)
			_, e = client.WaitForTransationReceipt(*lastHash, time.Second)
			util.OsExitIfErr(e, "Fail to get result of tx %+v", tx)

			// reset count and batch elem result
			rpcBatchElems = []clientRpc.BatchElem{}
			sendCount = 0
		}

		nonce = nonce.Add(nonce, big.NewInt(1))
	}

	e := os.Remove(resultPath)
	util.OsExitIfErr(e, "Remove result file error.")

	if failCount == 0 {
		e = os.Remove(warnPath)
		util.OsExitIfErr(e, "Remove result file error.")
	}

	fmt.Printf("Transfer done!\n")
}

func createTx(from types.Address, receiver Receiver, nonce *big.Int, chainID uint, epochHeight uint64) *types.UnsignedTransaction {
	tx := &types.UnsignedTransaction{}

	tx.From = &from
	tx.Gas = defaultGasLimit
	tx.GasPrice = types.NewBigIntByRaw(account.MustParsePrice())
	tx.StorageLimit = types.NewUint64(0)
	tx.ChainID = types.NewUint(chainID)
	tx.EpochHeight = types.NewUint64(epochHeight)

	tx.To = &receiver.Address
	tx.Nonce = types.NewBigIntByRaw(nonce)

	valueInBigInt := calcValue(receiveNumber, receiver.Weight)
	tx.Value = types.NewBigIntByRaw(valueInBigInt)

	return tx
}

func calcValue(numberPerTime decimal.Decimal, weigh decimal.Decimal) *big.Int {
	return receiveNumber.Mul(weigh).Mul(decimal.NewFromInt(1e18)).BigInt()
}

func mustParseInput() []Receiver {
	// read csv file
	content, err := ioutil.ReadFile(receiverListFile)
	util.OsExitIfErr(err, "read file %v error", receiverListFile)

	// parse to struct
	lines := strings.Split(string(content), "\n")
	receiverInfos := []Receiver{}
	for i, v := range lines {
		v = strings.Replace(v, "\t", " ", -1)
		v = strings.Replace(v, ",", " ", -1)
		items := strings.Fields(v)
		if len(v) == 0 {
			continue
		}

		if len(items) != 2 {
			util.OsExit("Line %v: %#v column number is %v, which shoule be 2\n", i, v, len(items))
		}

		weight, err := decimal.NewFromString(items[1])
		util.OsExitIfErr(err, "Parse %v to int error", weight)

		info := Receiver{
			Address: types.Address(items[0]),
			Weight:  weight,
		}
		receiverInfos = append(receiverInfos, info)
	}
	fmt.Printf("Receiver list count :%+v\n", len(receiverInfos))
	return receiverInfos
}

func initialEnviorment() (client *sdk.Client, am *sdk.AccountManager, lastPoint int, from types.Address, nonce *big.Int, chainID uint, epochHeight uint64) {

	am = account.DefaultAccountManager
	client = rpc.MustCreateClientWithRetry(100)
	client.SetAccountManager(am)

	from = types.Address(account.MustParseAccount())

	password := account.MustInputPassword("Enter password: ")
	am.Unlock(from, password)
	fmt.Println()

	// get inital Nonce
	nonce, e := client.GetNextNonce(from)
	util.OsExitIfErr(e, "Fail to get nonce of from %v", from)

	status, err := client.GetStatus()
	util.OsExitIfErr(err, "Fail to get status")
	chainID = uint(*status.ChainID)

	epoch, err := client.GetEpochNumber(types.EpochLatestState)
	util.OsExitIfErr(err, "Fail to get epoch")
	epochHeight = epoch.Uint64()

	lastPointStr, e := ioutil.ReadFile(resultPath)
	util.OsExitIfErr(e, "Fail to read result content")

	if len(lastPointStr) > 0 {
		lastPoint, e = strconv.Atoi(string(lastPointStr))
	} else {
		lastPoint = -1
	}
	return
}

func creatRecordFiles() (resultFs, warnFs *os.File) {
	resultFs, e := os.OpenFile(resultPath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0777)
	util.OsExitIfErr(e, "Failed to create file")
	// defer resultFs.Close()

	warnFs, e = os.OpenFile(warnPath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0777)
	util.OsExitIfErr(e, "Failed to create file")
	// defer warnFs.Close()
	return
}

func checkBalance(client *sdk.Client, from types.Address, receivers []Receiver) {
	balance, err := client.GetBalance(from)
	util.OsExitIfErr(err, "Failed to get balance")

	need := big.NewInt(0)
	for _, v := range receivers {
		// receiverNeed := big.NewInt(1).Mul(big.NewInt(int64(v.Weight*receiveNumber)), big.NewInt(1e18))
		receiverNeed := calcValue(receiveNumber, v.Weight)
		gasFee := big.NewInt(1).Mul(defaultGasLimit.ToInt(), account.MustParsePrice())
		need = need.Add(need, receiverNeed)
		need = need.Add(need, gasFee)
	}

	if balance.Cmp(need) < 0 {
		lastPointStr, e := ioutil.ReadFile(resultPath)
		util.OsExitIfErr(e, "Read result content error")

		if len(lastPointStr) == 0 {
			os.Remove(resultPath)
		}
		util.OsExit("Out of balance, need %v, has %v", util.DisplayValueWithUnit(need), util.DisplayValueWithUnit(balance))
	}
	fmt.Printf("Balance is enough, need %v, has %v\n", util.DisplayValueWithUnit(need), util.DisplayValueWithUnit(balance))
}
