package transfer

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Conflux-Chain/conflux-toolkit/account"
	"github.com/Conflux-Chain/conflux-toolkit/contract/common"
	"github.com/Conflux-Chain/conflux-toolkit/rpc"
	"github.com/Conflux-Chain/conflux-toolkit/types/cfxaddress"
	"github.com/Conflux-Chain/conflux-toolkit/util"
	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	clientRpc "github.com/Conflux-Chain/go-conflux-sdk/rpc"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/ethereum/go-ethereum/common/hexutil"

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

	resultFs := creatRecordFiles()
	defer func() {
		resultFs.Close()
		// warnFs.Close()
	}()

	receiverInfos := mustParseInput()
	client, am, lastPoint, from, nonce, chainID, epochHeight := initialEnviorment()

	// list cfx and ctoken for user select
	tokenSymbol, tokenAddress := selectToken()
	fmt.Printf("Selected token: %v, %v\n", tokenSymbol, tokenAddress)

	// check balance
	fmt.Println("===== Check if balance enough =====")
	checkBalance(client, from, receiverInfos, tokenAddress)

	sendCount := uint(0)
	rpcBatchElems := []clientRpc.BatchElem{}

	// transfer
	fmt.Println("===== Start batch transfer =====")
	for i, v := range receiverInfos {
		if i <= lastPoint {
			continue
		}

		tx := createTx(from, v, tokenAddress, nonce, chainID, epochHeight)
		err := client.ApplyUnsignedTransactionDefault(tx)
		util.OsExitIfErr(err, "Failed apply unsigned tx %+v", tx)

		// sign
		encoded, err := am.SignTransaction(*tx)
		util.OsExitIfErr(err, "Failed to sign transaction %+v", tx)
		fmt.Printf("%v. Sign send %v to %v with value %v done\n", i+1, tokenSymbol, v.Address,
			util.DisplayValueWithUnit(calcValue(receiveNumber, v.Weight)))

		// push to batch item array
		batchElemResult := types.Hash("")
		rpcBatchElems = append(rpcBatchElems, clientRpc.BatchElem{
			Method: "cfx_sendRawTransaction",
			Args:   []interface{}{"0x" + hex.EncodeToString(encoded)},
			Result: &batchElemResult,
		})

		sendCount++
		if sendCount == perBatchNum || i == len(receiverInfos)-1 {
			sendOneBatch(client, rpcBatchElems, i)
			sendCount = 0
		}
		nonce = nonce.Add(nonce, big.NewInt(1))
	}

	e := os.Remove(resultPath)
	util.OsExitIfErr(e, "Remove result file error.")
	fmt.Printf("===== Transfer done! =====\n")
}

func sendOneBatch(client *sdk.Client, rpcBatchElems []clientRpc.BatchElem, lastIndex int) {
	e := client.BatchCallRPC(rpcBatchElems)
	util.OsExitIfErr(e, "Batch send error")

	fails := []clientRpc.BatchElem{}
	for _, v := range rpcBatchElems {
		if v.Error != nil {
			fails = append(fails, v)
		}
	}
	if len(fails) > 0 {
		fmt.Printf("Fails details:%+v\n", fails)
	}

	// fmt.Printf("rpcBatchElems: %+v\n", rpcBatchElems)
	// save record
	ioutil.WriteFile(resultPath, []byte(strconv.Itoa(lastIndex)), 0777)
	// wait last packed
	lastHash := rpcBatchElems[len(rpcBatchElems)-1].Result.(*types.Hash)

	fmt.Printf("Batch send %v tx, total send %v done, failed %v, wait last be executed: %v\n", len(rpcBatchElems), lastIndex+1-len(fails), len(fails), lastHash)

	_, e = client.WaitForTransationReceipt(*lastHash, time.Second)
	util.OsExitIfErr(e, "Fail to get result of tx hash %+v", lastHash)
	// reset count and batch elem result
	rpcBatchElems = []clientRpc.BatchElem{}
}

func selectToken() (symbol string, contractAddress types.Address) {

	url := "https://confluxscan.io/v1/token?orderBy=transferCount&reverse=true&skip=0&limit=100&fields=price"

	req, _ := http.NewRequest("GET", url, nil)

	res, err := http.DefaultClient.Do(req)
	util.OsExitIfErr(err, "failed to get response by url %v", url)

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	util.OsExitIfErr(err, "failed to read token list from %v", res.Body)

	// fmt.Println(string(body))
	tokenList := struct {
		Total int `json:"total"`
		List  []struct {
			Address types.Address `json:"address"`
			Symbol  string        `json:"symbol"`
		} `json:"list"`
	}{}
	json.Unmarshal([]byte(body), &tokenList)

	// print token list for user select
	fmt.Println("These are the token list you could batch transfer:")
	fmt.Printf("%v. token: %v\n", 1, "CFX")
	for i := range tokenList.List {
		tokenList.List[i].Address = cfxaddress.FormatAddressToHex(tokenList.List[i].Address)
		fmt.Printf("%v. token: %v, contract address: %v\n", i+2, tokenList.List[i].Symbol, tokenList.List[i].Address)
	}

	selectedIdx := getSelectedIndex(len(tokenList.List) + 2)
	if selectedIdx == 1 {
		symbol = "CFX"
		return
	}
	token := tokenList.List[selectedIdx-2]
	if token.Symbol[0:1] != "c" {
		util.OsExit("Not support %v currently, please select token start with 'c', such as cUsdt, cMoon and so on.", token.Symbol)
	}
	return token.Symbol, token.Address
}

func getSelectedIndex(tokensCount int) int {
	// for loop until selected one
	fmt.Println("Please input the index you will transfer")
	selectedIdx := 0
	for {
		fmt.Scanln(&selectedIdx)
		if selectedIdx > 0 || selectedIdx <= tokensCount {
			fmt.Printf("You selected %v, press Y to continue, N to select again\n", selectedIdx)
			yes := "N"
			fmt.Scanln(&yes)
			if strings.ToUpper(yes) == "Y" {
				break
			}
			fmt.Printf("Please select again\n")
		}
		fmt.Printf("Input must be in range %v to %v, please input again\n", 1, tokensCount)
	}
	return selectedIdx
}

func createTx(from types.Address, receiver Receiver, token types.Address, nonce *big.Int, chainID uint, epochHeight uint64) *types.UnsignedTransaction {

	tx := &types.UnsignedTransaction{}

	tx.From = &from
	tx.GasPrice = types.NewBigIntByRaw(account.MustParsePrice())
	tx.ChainID = types.NewUint(chainID)
	tx.EpochHeight = types.NewUint64(epochHeight)
	tx.Nonce = types.NewBigIntByRaw(nonce)

	amountInDrip := calcValue(receiveNumber, receiver.Weight)
	if token == "" {
		tx.To = &receiver.Address
		tx.Value = types.NewBigIntByRaw(amountInDrip)
		tx.Gas = defaultGasLimit
		tx.StorageLimit = types.NewUint64(0)
	} else {
		tx.To = &token
		tx.Value = types.NewBigInt(0)
		tx.Data = getTransferData(token, receiver.Address, amountInDrip)
	}
	return tx
}

func getTransferData(contractAddress types.Address, reciever types.Address, amountInDrip *big.Int) (data hexutil.Bytes) {
	ctoken := common.MustGetCTokenContract(contractAddress.String())
	data, err := ctoken.GetData("send", reciever.ToCommonAddress(), amountInDrip, []byte{})
	util.OsExitIfErr(err, "failed to get data of send ctoken %v to %v amount %v", contractAddress, reciever, amountInDrip)
	return data
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

		addr := types.Address(items[0])
		if types.NormalAddress != addr.GetAddressType() {
			util.OsExit("Found unsupported address %v in line %v", addr, i)
		}

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
	err := am.Unlock(from, password)
	util.OsExitIfErr(err, "Fail to unlock account")
	fmt.Println("Account is unlocked")

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

func creatRecordFiles() (resultFs *os.File) {
	resultFs, e := os.OpenFile(resultPath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0777)
	util.OsExitIfErr(e, "Failed to create file")
	// defer resultFs.Close()

	// warnFs, e = os.OpenFile(warnPath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0777)
	// util.OsExitIfErr(e, "Failed to create file")
	// defer warnFs.Close()
	return
}

func checkBalance(client *sdk.Client, from types.Address, receivers []Receiver, token types.Address) {
	var balance *big.Int
	var err error
	if token.String() == "" {
		balance, err = client.GetBalance(from)
		util.OsExitIfErr(err, "Failed to get CFX balance of %v", from)
	} else {
		contract := common.MustGetCTokenContract(token.String())
		err = contract.Call(nil, &balance, "balanceOf", *from.ToCommonAddress())
		util.OsExitIfErr(err, "Failed to get token %v balance of %v", token, from)
	}

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
		msg := fmt.Sprintf("Out of balance, need %v, has %v", util.DisplayValueWithUnit(need), util.DisplayValueWithUnit(balance))
		// warnFs.WriteString(msg)
		util.OsExit(msg)
	}
	fmt.Printf("Balance is enough, need %v, has %v\n", util.DisplayValueWithUnit(need), util.DisplayValueWithUnit(balance))
}
