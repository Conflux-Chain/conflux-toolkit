package transfer

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Conflux-Chain/conflux-toolkit/account"
	"github.com/Conflux-Chain/conflux-toolkit/contract/common"
	"github.com/Conflux-Chain/conflux-toolkit/rpc"
	"github.com/sirupsen/logrus"

	"github.com/Conflux-Chain/conflux-toolkit/util"
	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	"github.com/Conflux-Chain/go-conflux-sdk/cfxclient/bulk"
	clientRpc "github.com/Conflux-Chain/go-conflux-sdk/rpc"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/Conflux-Chain/go-conflux-sdk/types/cfxaddress"
	"github.com/Conflux-Chain/go-conflux-sdk/utils"
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
	resultPath = "./transfer_result.json"

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

	// returns sended number
	// doOneBatch := func(elems []clientRpc.BatchElem, summay *BatchSummary) int {
	// 	oneBatchSummary := batchSendAndWaitReceipt(elems)
	// 	env.pState.saveSendings(env.pState.SendingStartIdx+len(elems), nil)

	// 	summay.Merge(oneBatchSummary)

	// 	fmt.Printf("===== Batch sent and executed %v tx done, failed %v =====\n", oneBatchSummary.total, oneBatchSummary.GetTotalFailCount())
	// 	if len(oneBatchSummary.failInfos) > 0 {
	// 		fmt.Printf("Fails details:\n%+v\n", strings.Join(oneBatchSummary.failInfos, "\n"))
	// 	}
	// 	return len(elems)
	// }

	fmt.Println("Initial enviorment")
	initialEnviorment()

	defer clearCacheFile()

	receiverInfos := mustParseReceivers()
	env.pState.refreshByReceivers(receiverInfos)

	// list cfx and ctoken for user select
	tokenSymbol, tokenAddress := selectToken()
	fmt.Printf("Selected token: %v, contract address: %v\n", tokenSymbol, tokenAddress)

	// transfer
	fmt.Println("===== Start batch transfer =====")
	batchSummary := BatchSummary{}

	// firstly send last send pendings
	env.pState.saveSelectToken(tokenSymbol, tokenAddress)
	receiverInfos = receiverInfos[env.pState.SendingStartIdx:]

	logrus.Debugf("a receiverInfos len:%v\n", len(receiverInfos))
	if len(env.pState.SendingBatchElems) > 0 {
		fmt.Printf("==== There are uncompleted tx in last time, start send one batch from %v and length %v ===\n",
			env.pState.SendingStartIdx, len(env.pState.SendingBatchElems))

		sentNum := sendOneBatch(env.pState.SendingBatchElems, &batchSummary)
		receiverInfos = receiverInfos[sentNum:]
		logrus.Debugf("aa receiverInfos len:%v\n", len(receiverInfos))
	}

	if len(receiverInfos) > 0 {
		// check balance
		fmt.Println("===== Check if balance enough =====")
		checkBalance(env.client, env.from, receiverInfos, tokenAddress, tokenSymbol)

		estimates := estimateGasAndCollateral(tokenAddress)
		for len(receiverInfos) > 0 {
			batchNum := int(math.Min(float64(perBatchNum), float64(len(receiverInfos))))

			// refresh nonce, because last batch may be has error like "tx pool is full", so refresh it
			// _nonce, err := env.client.GetNextNonce(env.from)
			// util.OsExitIfErr(err, "failed get nonce")
			// env.nonce = _nonce.ToInt()

			fmt.Printf("\n===== Start send one batch with %v tx=====\n", batchNum)
			elems := creatOneBatchElems(receiverInfos[:batchNum], tokenAddress, tokenSymbol, estimates)
			sentNum := sendOneBatch(elems, &batchSummary)
			receiverInfos = receiverInfos[sentNum:]
			logrus.Debugf("aaa receiverInfos len:%v\n", len(receiverInfos))
		}
	}

	fmt.Printf("\n===== All transfer done =====\n")
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
	password := "123" // account.MustInputPassword("Enter password: ")

	err = env.am.Unlock(env.from, password)
	util.OsExitIfErr(err, "Failed to unlock account")

	fmt.Printf("Account %v is unlocked\n", env.from)

	// get inital Nonce
	_nonce, e := env.client.TxPool().NextNonce(env.from)
	env.nonce = _nonce.ToInt()
	util.OsExitIfErr(e, "Failed to get nonce of from %v", env.from)

	epoch, err := env.client.GetEpochNumber(types.EpochLatestState)
	util.OsExitIfErr(err, "Failed to get epoch")
	env.epochHeight = epoch.ToInt().Uint64()

	env.pState = loadProcessState()
	// if _, err := os.Stat(resultPath); os.IsNotExist(err) {
	// 	env.processState.SendingsStartIdx = nil
	// 	return
	// }

	// lastPointStr, e := ioutil.ReadFile(resultPath)
	// util.OsExitIfErr(e, "Failed to read result content")

	// if len(lastPointStr) > 0 {
	// 	env.processState.SendingsStartIdx, e = strconv.Atoi(string(lastPointStr))
	// 	util.OsExitIfErr(e, "Failed to parse result content")
	// } else {
	// 	env.SendingsStartIdx = -1
	// }
}

func sendOneBatch(elems []clientRpc.BatchElem, summay *BatchSummary) int {
	oneBatchSummary := batchSendAndWaitReceipt(elems)
	env.pState.saveSendings(env.pState.SendingStartIdx+len(elems), nil)

	summay.Merge(oneBatchSummary)

	fmt.Printf("===== Batch sent and executed %v tx done, failed %v =====\n", oneBatchSummary.total, oneBatchSummary.GetTotalFailCount())
	if len(oneBatchSummary.failInfos) > 0 {
		fmt.Printf("Fails details:\n%+v\n", strings.Join(oneBatchSummary.failInfos, "\n"))
	}
	return len(elems)
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
	em, err := env.client.EstimateGasAndCollateral(callReq)
	util.OsExitIfErr(err, "failed get estimate of %v", callReq)

	// double gasLimit to avoid transaction fail because of "out of gas"
	_doubledGasLimit := new(big.Int).Mul(em.GasLimit.ToInt(), big.NewInt(2))
	if _doubledGasLimit.Cmp(big.NewInt(15000000)) > 0 {
		_doubledGasLimit = big.NewInt(15000000)
	}
	em.GasLimit = types.NewBigIntByRaw(_doubledGasLimit)
	return em
}

func creatOneBatchElems(oneBatchReceiver []Receiver, tokenAddress *cfxaddress.Address, tokenSymbol string, estimates types.Estimate) (elems []clientRpc.BatchElem) {
	// env.lastPoint is start with -1, so env.lastPoint+1 is actual index, env.lastPoint + 2 is the index count from 1
	startCnt := env.pState.SendingStartIdx + 1 //- len(oneBatchReceiver)

	rpcBatchElems := []clientRpc.BatchElem{}
	for i, v := range oneBatchReceiver {
		tx := createTx(env.from, v, tokenAddress, env.nonce, estimates)
		rpcBatchElems = append(rpcBatchElems, createBatchElemItem(tx))

		fmt.Printf("%v. Sign send %v to %v with value %v nonce %v done\n",
			startCnt+i, tokenSymbol, cfxaddress.MustNew(v.Address, env.networkID),
			util.DisplayValueWithUnit(calcValue(weight, v.AmountInCfx)), tx.Nonce)
		env.nonce = env.nonce.Add(env.nonce, big.NewInt(1))
	}
	return rpcBatchElems
}

func createBatchElemItem(tx *types.UnsignedTransaction) clientRpc.BatchElem {
	err := env.client.ApplyUnsignedTransactionDefault(tx)
	util.OsExitIfErr(err, "Failed apply unsigned tx %+v", tx)

	// sign
	encoded, err := env.am.SignTransaction(*tx)
	util.OsExitIfErr(err, "Failed to sign transaction %+v", tx)

	// fmt.Printf("%v. Sign send %v to %v with value %v nonce %v done\n",
	// 	startCnt+i, tokenSymbol, cfxaddress.MustNew(v.Address, env.networkID),
	// 	util.DisplayValueWithUnit(calcValue(weight, v.AmountInCfx)), tx.Nonce)

	// push to batch item array
	batchElemResult := new(string)
	return clientRpc.BatchElem{
		Method: "cfx_sendRawTransaction",
		Args:   []interface{}{"0x" + hex.EncodeToString(encoded)},
		Result: batchElemResult,
	}
}

func batchSendAndWaitReceipt(rpcBatchElems []clientRpc.BatchElem) BatchSummary {
	summary := BatchSummary{
		total: len(rpcBatchElems),
	}

	batchSend(rpcBatchElems, nil)
	waitLastReceipt(rpcBatchElems)
	batchGetReceipts(rpcBatchElems, &summary)

	return summary
}

func batchSend(rpcBatchElems []clientRpc.BatchElem, needSends []bool) {
	needSends, allSendDone := refreshNeedSends(rpcBatchElems, needSends)
	logrus.Debugf("needSends: %v, allSendDone %v\n", needSends, allSendDone)
	if allSendDone {
		return
	}

	// wait response
	hashDoneChan := util.WaitSigAndPrintDot()
	e := env.client.BatchCallRPC(rpcBatchElems)
	hashDoneChan <- nil
	util.OsExitIfErr(e, "Batch send error")
	fmt.Println("\n== Received tx hash list")

	env.pState.saveSendings(env.pState.SendingStartIdx, rpcBatchElems)

	for i, v := range rpcBatchElems {
		posOfAll := env.pState.SendingStartIdx + 1 + i //- len(rpcBatchElems)

		if !needSends[i] {
			continue
		}

		if v.Error == nil {
			needSends[i] = false
			fmt.Printf("%v. txhash %v \n", posOfAll, *(v.Result.(*string)))
			continue
		}

		rpcError, e := utils.ToRpcError(v.Error)
		if e == nil {
			v.Error = rpcError
		} else {
			fmt.Printf("not a valid rpc error , Type %t, %v\n", v.Error, v.Error)
		}

		fmt.Printf("%v. send error %v, will auto re-send later \n", posOfAll, rpcError)

		// regenerate tx for errored tx, the tx error must be "tx is full" or "tx already exist"
		var err error
		if rpcBatchElems[i], err = reorgnizeTx(v); err != nil {
			util.OsExitIfErr(err, "%v. Failed to reorgnize tx %v\n", posOfAll)
		}
	}
	// re-send
	time.Sleep(time.Second * 2)
	batchSend(rpcBatchElems, needSends)
}

func refreshNeedSends(rpcBatchElems []clientRpc.BatchElem, needSends []bool) (populated []bool, allSendDone bool) {
	// No result or error is not nil, need send
	if needSends == nil {
		needSends = make([]bool, len(rpcBatchElems))
		for i := range needSends {
			result := rpcBatchElems[i].Result.(*string)
			needSends[i] = rpcBatchElems[i].Error != nil || *result == ""
			// fmt.Printf("rpcBatchElems[i].Error == %v, result= %v \n", rpcBatchElems[i].Error, *result)
		}
	}

	if len(rpcBatchElems) != len(needSends) {
		util.OsExit("batch elem length must equal to isSend flgas length")
	}

	// if no need send, return
	allSendDone = true
	for _, v := range needSends {
		if v {
			allSendDone = false
			break
		}
	}
	return needSends, allSendDone
}

func reorgnizeTx(be clientRpc.BatchElem) (clientRpc.BatchElem, error) {
	rawTxStr := be.Args[0].(string)
	rawTxByts, err := hex.DecodeString(rawTxStr[2:])
	util.OsExitIfErr(err, "Failed to decode raw tx string %v", rawTxStr)

	signedTx := &types.SignedTransaction{}
	err = signedTx.Decode(rawTxByts, env.networkID)
	util.OsExitIfErr(err, "%v. Failed to decode signed tx %v\n")

	signedTx.UnsignedTransaction.GasPrice = types.NewBigIntByRaw(new(big.Int).Add(signedTx.UnsignedTransaction.GasPrice.ToInt(), big.NewInt(1)))
	reorgedItem := createBatchElemItem(&signedTx.UnsignedTransaction)

	fmt.Printf("Reorgnized tx with new gas price %v\n", signedTx.UnsignedTransaction.GasPrice.ToInt())
	logrus.Debugf("Reorgnized tx %+v with new gas price\n", signedTx)
	return reorgedItem, nil
}

func waitLastReceipt(rpcBatchElems []clientRpc.BatchElem) {
	// wait last be packed
	var lastHash *types.Hash
	for i := len(rpcBatchElems); i > 0; i-- {
		if rpcBatchElems[i-1].Error == nil {
			lastHash = (*types.Hash)(rpcBatchElems[i-1].Result.(*string))
			break
		}
	}
	// all are error, return
	if lastHash == nil {
		fmt.Println("Failed to send all of this batch of transactions")
		return
	}

	fmt.Printf("\nBatch sent %v, wait last valid tx hash be executed: %v ", len(rpcBatchElems), lastHash)

	receiptDoneChan := util.WaitSigAndPrintDot()
	_, e := env.client.WaitForTransationReceipt(*lastHash, time.Second)
	receiptDoneChan <- nil
	if e != nil {
		util.OsExitIfErr(e, "Failed to get receipt of %+v", lastHash)
	}

	fmt.Printf(" executed! \n\n")
}

func batchGetReceipts(rpcBatchElems []clientRpc.BatchElem, summary *BatchSummary) {
	// check if all transaction executed successfully
	bulkCaller := bulk.NewBulkCaller(env.client)

	failInfos := make([]string, 0)
	txReceipts := make([]*types.TransactionReceipt, len(rpcBatchElems))
	receiptErrors := make([]*error, len(rpcBatchElems))

	// all send tx and get tx receipt errors, rpcBatchElems.Error is sent tx error,
	// and GetTransactionReceipt error also be saved to allErrors
	allErrors := make([]error, len(rpcBatchElems))

	receiptErrIdxInAll := make([]int, 0)
	for i, v := range rpcBatchElems {
		allErrors[i] = v.Error
		if v.Error != nil {
			continue
		}

		txHash := (*types.Hash)(v.Result.(*string))
		txReceipts[i], receiptErrors[i] = bulkCaller.GetTransactionReceipt(*txHash)
		receiptErrIdxInAll = append(receiptErrIdxInAll, i)
	}

	err := bulkCaller.Execute()
	if err != nil {
		util.OsExitIfErr(err, "Failed to request transaction receipts: %+v", err)
	}

	for i, v := range receiptErrors {
		if v != nil {
			allErrors[receiptErrIdxInAll[i]] = *v
		}
	}

	for i, r := range txReceipts {
		posOfAll := env.pState.SendingStartIdx + 1 + i //- len(rpcBatchElems)
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
	networkId, err := env.client.GetNetworkID()
	util.OsExitIfErr(err, "Failed to get networkID")

	url := "https://confluxscan.io/v1/token?orderBy=transferCount&reverse=true&skip=0&limit=100&fields=price"
	if networkId == util.TESTNET {
		url = "https://testnet.confluxscan.io/v1/token?orderBy=transferCount&reverse=true&skip=0&limit=100&fields=price"
	}

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

	selectedIdx := 1 // getSelectedIndex(len(tokenList.List) + 2)
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

	tx.Gas = estimates.GasLimit
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

func checkBalance(client *sdk.Client, from types.Address, receivers []Receiver, token *types.Address, tokenSymbol string) {

	var (
		cfxBalance   *big.Int = big.NewInt(0)
		tokenBalance *big.Int = big.NewInt(0)

		perTxGasNeed     *big.Int = new(big.Int).Mul(account.MustParsePrice(), defaultGasLimit.ToInt())
		perTxStorageNeed *big.Int = big.NewInt(0)

		receiveNeed *big.Int = big.NewInt(0)
		gasNeed     *big.Int = big.NewInt(0)
		storageNeed *big.Int = big.NewInt(0)
	)

	_balance, err := client.GetBalance(from)
	cfxBalance = _balance.ToInt()
	util.OsExitIfErr(err, "Failed to get CFX balance of %v", from)

	if token != nil {
		contract := common.MustGetCTokenContract(token.String())
		err := contract.Call(nil, &tokenBalance, "balanceOf", from.MustGetCommonAddress())
		util.OsExitIfErr(err, "Failed to get token %v balance of %v", tokenSymbol, from)

		_price := (*hexutil.Big)(account.MustParsePrice())
		em := estimateGasAndCollateral(token)
		aginstResp, err := client.CheckBalanceAgainstTransaction(from, *token, em.GasLimit, _price, em.StorageCollateralized)
		util.OsExitIfErr(err, "Failed to check balance against tx")

		// needPayGas = aginstResp.WillPayTxFee
		// needPayStorage = aginstResp.WillPayCollateral
		if aginstResp.WillPayTxFee {
			perTxGasNeed = new(big.Int).Mul(account.MustParsePrice(), em.GasLimit.ToInt())
		}
		if aginstResp.WillPayCollateral {
			perTxStorageNeed = new(big.Int).Mul(account.MustParsePrice(), em.StorageCollateralized.ToInt())
		}
	}

	for _, v := range receivers {
		receiverNeed := calcValue(weight, v.AmountInCfx)
		// gasFee := big.NewInt(0)
		// if token == nil {
		// 	gasFee = big.NewInt(1).Mul(defaultGasLimit.ToInt(), account.MustParsePrice())
		// }

		receiveNeed = receiveNeed.Add(receiveNeed, receiverNeed)
		gasNeed = gasNeed.Add(gasNeed, perTxGasNeed)
		storageNeed = storageNeed.Add(storageNeed, perTxStorageNeed)
	}

	if token == nil {
		cfxNeed := big.NewInt(0).Add(receiveNeed, gasNeed)
		cfxNeed = big.NewInt(0).Add(cfxNeed, storageNeed)
		if cfxBalance.Cmp(cfxNeed) < 0 {
			// clearCacheFile()
			msg := fmt.Sprintf("Balance of %v is not enough, need %v, has %v",
				from, util.DisplayValueWithUnit(receiveNeed), util.DisplayValueWithUnit(cfxBalance))
			util.OsExit(msg)
		}
	} else {
		cfxNeed := big.NewInt(0).Add(gasNeed, storageNeed)
		if cfxBalance.Cmp(cfxNeed) < 0 || tokenBalance.Cmp(receiveNeed) < 0 {
			// clearCacheFile()
			msg := fmt.Sprintf("Token %v balance of %v is not enough or CFX balance not enough to pay gas,"+
				"%v need %v, has %v, CFX need %v, has %v",
				tokenSymbol, from,
				tokenSymbol, util.DisplayValueWithUnit(receiveNeed, tokenSymbol), util.DisplayValueWithUnit(tokenBalance, tokenSymbol),
				util.DisplayValueWithUnit(cfxNeed), util.DisplayValueWithUnit(cfxBalance),
			)
			util.OsExit(msg)
		}
	}

	fmt.Printf("Balance of %v is enough, need %v, has %v\n", from, util.DisplayValueWithUnit(receiveNeed, tokenSymbol), util.DisplayValueWithUnit(cfxBalance, tokenSymbol))
}
