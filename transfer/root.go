package transfer

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Conflux-Chain/conflux-toolkit/account"
	"github.com/Conflux-Chain/conflux-toolkit/contract/common"
	"github.com/Conflux-Chain/conflux-toolkit/rpc"

	"github.com/Conflux-Chain/conflux-toolkit/util"
	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	"github.com/Conflux-Chain/go-conflux-sdk/cfxclient/bulk"
	clientRpc "github.com/Conflux-Chain/go-conflux-sdk/rpc"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/Conflux-Chain/go-conflux-sdk/types/cfxaddress"
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
	resultPath = "./transfer_result.txt"

	defaultGasLimit = types.NewBigInt(21000)

	// command flags
	receiverListFile string
	weight           decimal.Decimal

	perBatchNum uint
	env         enviorment
)

func init() {
	rpc.AddURLVar(rootCmd)
	account.AddFromVar(rootCmd)
	account.AddGasPriceVar(rootCmd)

	rootCmd.PersistentFlags().StringVar(&receiverListFile, "receivers", "", "receiver list file path")
	rootCmd.MarkPersistentFlagRequired("receivers")

	// the weight of send, the actual send amount is weight * amount
	sentWeightInStr := ""
	rootCmd.PersistentFlags().StringVar(&sentWeightInStr, "weight", "1", "send weight, the actual send amount is weight * amount")
	rootCmd.MarkPersistentFlagRequired("weight")

	rootCmd.PersistentFlags().UintVar(&perBatchNum, "batch", 1000, "send tx number per batch")
	formatReceiverNumber(sentWeightInStr)
}

func formatReceiverNumber(sentWeightInStr string) {
	var err error
	weight, err = decimal.NewFromString(sentWeightInStr)
	util.OsExitIfErr(err, "receiveNumber %v is not a number", sentWeightInStr)
}

// SetParent sets parent command
func SetParent(parent *cobra.Command) {
	parent.AddCommand(rootCmd)
}

func doTransfers(cmd *cobra.Command, args []string) {
	fmt.Println("Initial enviorment")
	initialEnviorment()

	resultFs := creatRecordFiles()
	defer func() {
		resultFs.Close()
		e := os.Remove(resultPath)
		util.OsExitIfErr(e, "Remove result file error.")
	}()

	receiverInfos := mustParseReceivers()

	// list cfx and ctoken for user select
	tokenSymbol, tokenAddress := selectToken()
	fmt.Printf("Selected token: %v, contract address: %v\n", tokenSymbol, tokenAddress)

	estimates := estimateGasAndCollateral(tokenAddress)

	// check balance
	fmt.Println("===== Check if balance enough =====")
	checkBalance(env.client, env.from, receiverInfos, tokenAddress, tokenSymbol)

	// transfer
	fmt.Println("===== Start batch transfer =====")
	receiverInfos = receiverInfos[(env.lastPoint + 1):]
	batchSummary := BatchSummary{}
	for len(receiverInfos) > 0 {
		batchNum := int(math.Min(float64(perBatchNum), float64(len(receiverInfos))))
		// startPosition := env.lastPoint
		env.lastPoint += batchNum

		elems := creatOneBatchElems(receiverInfos[:batchNum], tokenAddress, tokenSymbol, estimates)
		oneBatchSummary := sendOneBatchAndWaitReceipt(elems)
		batchSummary.Merge(oneBatchSummary)
		receiverInfos = receiverInfos[len(elems):]
	}

	fmt.Printf("===== All transfer done =====\n")
	fmt.Printf("Summares:\n%v\n", &batchSummary)

	fmt.Printf("===== Complete! =====\n")
}

func initialEnviorment() {

	env.am = account.DefaultAccountManager
	env.client = rpc.MustCreateClientWithRetry(100)
	env.client.SetAccountManager(env.am)

	status, err := env.client.GetStatus()
	util.OsExitIfErr(err, "Failed to get status")

	env.chainID = uint32(status.ChainID)
	env.networkID = uint32(status.NetworkID)

	env.from = cfxaddress.MustNew(account.MustParseAccount().GetHexAddress(), env.networkID)
	password := account.MustInputPassword("Enter password: ")

	err = env.am.Unlock(env.from, password)
	util.OsExitIfErr(err, "Failed to unlock account")

	fmt.Printf("Account %v is unlocked\n", env.from)

	// get inital Nonce
	_nonce, e := env.client.GetNextNonce(env.from)
	env.nonce = _nonce.ToInt()
	util.OsExitIfErr(e, "Failed to get nonce of from %v", env.from)

	epoch, err := env.client.GetEpochNumber(types.EpochLatestState)
	util.OsExitIfErr(err, "Failed to get epoch")
	env.epochHeight = epoch.ToInt().Uint64()

	if _, err := os.Stat(resultPath); os.IsNotExist(err) {
		env.lastPoint = -1
		return
	}

	lastPointStr, e := ioutil.ReadFile(resultPath)
	util.OsExitIfErr(e, "Failed to read result content")

	if len(lastPointStr) > 0 {
		env.lastPoint, e = strconv.Atoi(string(lastPointStr))
		util.OsExitIfErr(e, "Failed to parse result content")
	} else {
		env.lastPoint = -1
	}
}

func estimateGasAndCollateral(tokenAddress *cfxaddress.Address) types.Estimate {
	if tokenAddress == nil {
		return types.Estimate{
			GasLimit:              defaultGasLimit,
			GasUsed:               defaultGasLimit,
			StorageCollateralized: types.NewBigInt(0),
		}
	}
	data := getTransferData(*tokenAddress, env.from, big.NewInt(0)).String()
	callReq := types.CallRequest{
		To:   tokenAddress,
		Data: &data,
	}
	sm, err := env.client.EstimateGasAndCollateral(callReq)
	util.OsExitIfErr(err, "failed get estimate of %v", callReq)
	return sm
}

func creatOneBatchElems(oneBatchReceiver []Receiver, tokenAddress *cfxaddress.Address, tokenSymbol string, estimates types.Estimate) (elems []clientRpc.BatchElem) {
	// env.lastPoint is start with -1, so env.lastPoint+1 is actual index, env.lastPoint + 2 is the index count from 1
	startCnt := env.lastPoint + 2 - len(oneBatchReceiver)

	rpcBatchElems := []clientRpc.BatchElem{}
	for i, v := range oneBatchReceiver {
		tx := createTx(env.from, v, tokenAddress, env.nonce, estimates)
		err := env.client.ApplyUnsignedTransactionDefault(tx)
		util.OsExitIfErr(err, "Failed apply unsigned tx %+v", tx)

		// sign
		encoded, err := env.am.SignTransaction(*tx)
		util.OsExitIfErr(err, "Failed to sign transaction %+v", tx)

		fmt.Printf("%v. Sign send %v to %v with value %v done\n", startCnt+i, tokenSymbol, cfxaddress.MustNew(v.Address, env.networkID),
			util.DisplayValueWithUnit(calcValue(weight, v.AmountInCfx)))

		// push to batch item array
		batchElemResult := types.Hash("")
		rpcBatchElems = append(rpcBatchElems, clientRpc.BatchElem{
			Method: "cfx_sendRawTransaction",
			Args:   []interface{}{"0x" + hex.EncodeToString(encoded)},
			Result: &batchElemResult,
		})
		env.nonce = env.nonce.Add(env.nonce, big.NewInt(1))
	}
	return rpcBatchElems
}

func sendOneBatchAndWaitReceipt(rpcBatchElems []clientRpc.BatchElem) BatchSummary {
	summary := BatchSummary{
		total: len(rpcBatchElems),
	}

	batchSend(rpcBatchElems)
	// save record
	ioutil.WriteFile(resultPath, []byte(strconv.Itoa(env.lastPoint)), 0777)

	waitLastReceipt(rpcBatchElems)
	batchGetReceipts(rpcBatchElems, &summary)

	fmt.Printf("\nBatch sent %v tx, failed %v\n", len(rpcBatchElems), summary.GetTotalFailCount())
	if len(summary.failInfos) > 0 {
		fmt.Printf("Fails details:%+v\n", strings.Join(summary.failInfos, "\n"))
	}
	return summary
}

func batchSend(rpcBatchElems []clientRpc.BatchElem) {
	// wait response
	fmt.Printf("\nSending the batch transactions, please wait response")

	hashDoneChan := util.WaitSigAndPrintDot()
	e := env.client.BatchCallRPC(rpcBatchElems)
	hashDoneChan <- nil
	fmt.Printf("\nReceived send response\n")
	util.OsExitIfErr(e, "Batch send error")
}

func waitLastReceipt(rpcBatchElems []clientRpc.BatchElem) {
	// wait last be packed
	var lastHash *types.Hash
	for i := len(rpcBatchElems); i > 0; i-- {
		// ok := false
		// lastHash, ok = rpcBatchElems[i-1].Result.(*types.Hash)
		// fmt.Printf("rpcBatchElems[i-1]%v,lasthash: %v\n", rpcBatchElems[i-1], lastHash)
		// if ok && lastHash != nil {
		// 	break
		// }

		if rpcBatchElems[i-1].Error == nil {
			lastHash = rpcBatchElems[i-1].Result.(*types.Hash)
			break
		}
	}
	// all are error, return
	if lastHash == nil {
		fmt.Println("Failed to send all of this batch of transactions")
		return
	}

	fmt.Printf("\nBatch sent %v, wait last valid tx hash be executed: %v \n", len(rpcBatchElems), lastHash)

	receiptDoneChan := util.WaitSigAndPrintDot()
	_, e := env.client.WaitForTransationReceipt(*lastHash, time.Second)
	receiptDoneChan <- nil
	if e != nil {
		util.OsExitIfErr(e, "Failed to get result of last valid tx hash %+v", lastHash)
	}

	fmt.Printf("\nThe last valid tx %v of this batch is executed\n\n", lastHash)
}

func batchGetReceipts(rpcBatchElems []clientRpc.BatchElem, summary *BatchSummary) {
	// check if all transaction executed successfully
	bulkCaller := bulk.NewBulkerCaller(env.client)

	failInfos := make([]string, 0)
	txReceipts := make([]*types.TransactionReceipt, len(rpcBatchElems))
	allErrors := make([]error, len(rpcBatchElems))

	receiptErrIdxInAll := make([]int, 0)
	for i, v := range rpcBatchElems {
		allErrors[i] = v.Error
		if v.Error != nil {
			continue
		}
		txReceipts[i] = bulkCaller.GetTransactionReceipt(*v.Result.(*types.Hash))
		receiptErrIdxInAll = append(receiptErrIdxInAll, i)
	}

	receiptErrors, err := bulkCaller.Excute()
	// fmt.Printf("erros len %v\n", len(receiptErrors))
	// fmt.Printf("allErrors len %v\n", len(allErrors))
	if err != nil {
		util.OsExitIfErr(err, "Failed to request transaction receipts: %+v", err)
	}

	for i, v := range receiptErrors {
		allErrors[receiptErrIdxInAll[i]] = v
	}

	for i, r := range txReceipts {
		posOfAll := env.lastPoint + 2 - len(rpcBatchElems) + i
		if r != nil && r.OutcomeStatus == 0 {
			continue
		}

		if rpcBatchElems[i].Error != nil {
			failInfos = append(failInfos, failSentTx(posOfAll, allErrors[i]))
			summary.sentTxfailedCount++
			continue
		}

		if allErrors[i] != nil {
			failInfos = append(failInfos, failGetTxReceipt(posOfAll, allErrors[i]))
			summary.getReceiptFailedCount++
			continue
		}

		// In normal case, the transaction receipt could not nil when tx is executed, so it's impossible to be nil here, but just record it if really happens
		if r == nil {
			failInfos = append(failInfos, failTxReceiptNull(posOfAll))
			summary.receiptNullCount++
			continue
		}

		if r.OutcomeStatus != 0 {
			failInfos = append(failInfos, failExecuteTx(posOfAll, r.TxExecErrorMsg))
			summary.executFailedCount++
			continue
		}
	}
	summary.failInfos = failInfos
}

func selectToken() (symbol string, contractAddress *types.Address) {

	url := "https://confluxscan.io/v1/token?orderBy=transferCount&reverse=true&skip=0&limit=100&fields=price"

	req, _ := http.NewRequest("GET", url, nil)

	res, err := http.DefaultClient.Do(req)
	util.OsExitIfErr(err, "Failed to get response by url %v", url)

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	util.OsExitIfErr(err, "Failed to read token list from %v", res.Body)

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
		fmt.Printf("%v. token: %v, contract address: %v\n", i+2, tokenList.List[i].Symbol, tokenList.List[i].Address)
	}

	selectedIdx := getSelectedIndex(len(tokenList.List) + 2)
	if selectedIdx == 1 {
		symbol = "CFX"
		return
	}
	token := tokenList.List[selectedIdx-2]
	if token.Symbol != "FC" && token.Symbol[0:1] != "c" {
		util.OsExit("Not support %v currently, please select token FC or starts with 'c', such as cUsdt, cMoon and so on.", token.Symbol)
	}
	return token.Symbol, &token.Address
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

func createTx(from types.Address, receiver Receiver, token *types.Address, nonce *big.Int, estimates types.Estimate) *types.UnsignedTransaction {
	tx := &types.UnsignedTransaction{}

	tx.From = &from
	tx.GasPrice = types.NewBigIntByRaw(account.MustParsePrice())
	tx.ChainID = types.NewUint(uint(env.chainID))
	tx.EpochHeight = types.NewUint64(env.epochHeight)
	tx.Nonce = types.NewBigIntByRaw(nonce)

	amountInDrip := calcValue(weight, receiver.AmountInCfx)
	to := cfxaddress.MustNew(receiver.Address, env.chainID)
	if token == nil {
		tx.To = &to
		tx.Value = types.NewBigIntByRaw(amountInDrip)
		tx.StorageLimit = types.NewUint64(0)
	} else {
		tx.To = token
		tx.Value = types.NewBigInt(0)
		tx.Data = getTransferData(*token, to, amountInDrip)
	}

	// double gasPrice to avoid transaction fail because of "out of gas"
	_gasLimit := new(big.Int).Mul(estimates.GasLimit.ToInt(), big.NewInt(2))
	if _gasLimit.Cmp(big.NewInt(15000000)) > 0 {
		_gasLimit = big.NewInt(15000000)
	}

	// _gasLimit = estimates.GasLimit.ToInt().Mul(estimates.GasLimit.ToInt(), big.NewInt(2))
	// if tx.Nonce.String()[4] == '1' {
	// 	_gasLimit = big.NewInt(16000000000)
	// }

	// fmt.Printf("estimates.GasLimit %v, _gasLimit %v\n", estimates.GasLimit, _gasLimit)
	tx.Gas = (*hexutil.Big)(_gasLimit)
	tx.StorageLimit = types.NewUint64(estimates.StorageCollateralized.ToInt().Uint64())

	return tx
}

func getTransferData(contractAddress types.Address, reciever types.Address, amountInDrip *big.Int) (data hexutil.Bytes) {
	ctoken := common.MustGetCTokenContract(contractAddress.String())
	data, err := ctoken.GetData("send", reciever.MustGetCommonAddress(), amountInDrip, []byte{})
	util.OsExitIfErr(err, "Failed to get data of send ctoken %v to %v amount %v", contractAddress, reciever, amountInDrip)
	return data
}

func calcValue(numberPerTime decimal.Decimal, weigh decimal.Decimal) *big.Int {
	return weight.Mul(weigh).Mul(decimal.NewFromInt(1e18)).BigInt()
}

func mustParseReceivers() []Receiver {
	// read csv file
	content, err := ioutil.ReadFile(receiverListFile)
	util.OsExitIfErr(err, "Failed to read file %v", receiverListFile)

	// parse to struct
	lines := strings.Split(string(content), "\n")
	receiverInfos := []Receiver{}

	invalids := []string{}

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

		isDecimal, err := regexp.Match(`^\d+\.?\d*$`, []byte(items[1]))
		util.OsExitIfErr(err, "Failed to regex check %v ", items[1])
		if !isDecimal {
			invalids = append(invalids, fmt.Sprintf("Line %v: Number %v is unsupported, only supports pure number format, scientific notation like 1e18 and other representation format are unspported", i+1, items[1]))
			continue
		}

		amountInCfx, err := decimal.NewFromString(items[1])
		if err != nil {
			invalids = append(invalids, fmt.Sprintf("Line %v: Failed to parse %v to int, errmsg:%v", i+1, items[1], err.Error()))
			continue
		}

		_, err = cfxaddress.New(items[0])
		if err != nil {
			invalids = append(invalids, fmt.Sprintf("Line %v: Failed to create cfx address by %v, Errmsg: %v", i+1, items[0], err.Error()))
			continue
		}

		info := Receiver{
			Address:     items[0],
			AmountInCfx: amountInCfx,
		}

		receiverInfos = append(receiverInfos, info)
	}

	if len(invalids) > 0 {
		errMsg := fmt.Sprintf("Invalid Recevier info exists:\n%v", strings.Join(invalids, "\n"))
		util.OsExit(errMsg)
	}

	fmt.Printf("Receiver list count :%+v\n", len(receiverInfos))
	return receiverInfos
}

func creatRecordFiles() (resultFs *os.File) {
	resultFs, e := os.OpenFile(resultPath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0777)
	util.OsExitIfErr(e, "Failed to create file")
	return
}

func checkBalance(client *sdk.Client, from types.Address, receivers []Receiver, token *types.Address, tokenSymbol string) {
	var balance *big.Int
	var err error
	if token == nil {
		_balance, err := client.GetBalance(from)
		balance = _balance.ToInt()
		util.OsExitIfErr(err, "Failed to get CFX balance of %v", from)
	} else {
		contract := common.MustGetCTokenContract(token.String())
		err = contract.Call(nil, &balance, "balanceOf", from.MustGetCommonAddress())
		util.OsExitIfErr(err, "Failed to get token %v balance of %v", tokenSymbol, from)
	}

	need := big.NewInt(0)
	for _, v := range receivers {
		receiverNeed := calcValue(weight, v.AmountInCfx)
		gasFee := big.NewInt(0)
		if token == nil {
			gasFee = big.NewInt(1).Mul(defaultGasLimit.ToInt(), account.MustParsePrice())
		}
		// TODO: check if sponsor when token transfer

		need = need.Add(need, receiverNeed)
		need = need.Add(need, gasFee)
	}

	if balance.Cmp(need) < 0 {
		lastPointStr, e := ioutil.ReadFile(resultPath)
		util.OsExitIfErr(e, "Read result content error")

		if len(lastPointStr) == 0 {
			os.Remove(resultPath)
		}
		msg := fmt.Sprintf("Out balance of %v, need %v, has %v", from, util.DisplayValueWithUnit(need, tokenSymbol), util.DisplayValueWithUnit(balance, tokenSymbol))
		util.OsExit(msg)
	}
	fmt.Printf("Balance of %v is enough, need %v, has %v\n", from, util.DisplayValueWithUnit(need, tokenSymbol), util.DisplayValueWithUnit(balance, tokenSymbol))
}

// ======= Fail Message =======
func failSentTx(i int, err error) string {
	return fmt.Sprintf("The %vth transaction failed, Failed type: %v, Error Info: %+v\n", i, "send tx", err)
}

func failGetTxReceipt(i int, err error) string {
	return fmt.Sprintf("The %vth transaction failed, Failed type: %v, Error Info: %+v\n", i, "get tx receipt", err)
}

func failTxReceiptNull(i int) string {
	return fmt.Sprintf("The %vth transaction failed, Failed type: %v", i, "tx receipt null")
}

func failExecuteTx(i int, errMsg *string) string {
	return fmt.Sprintf("The %vth transaction failed, Failed type: %v, Error Info: %+v\n", i, "tx execute failed", util.GetStringVal(errMsg))
}

// func getPosOfAll(startPosition int, delta int) int {
// 	return startPosition + 2 + delta
// }
