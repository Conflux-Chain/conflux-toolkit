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
	"reflect"
	"runtime/debug"
	"strings"
	"time"

	"github.com/Conflux-Chain/conflux-toolkit/account"
	"github.com/Conflux-Chain/conflux-toolkit/contract/common"
	"github.com/Conflux-Chain/conflux-toolkit/rpc"
	"github.com/sirupsen/logrus"

	"github.com/Conflux-Chain/conflux-toolkit/util"
	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	"github.com/Conflux-Chain/go-conflux-sdk/cfxclient/bulk"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/Conflux-Chain/go-conflux-sdk/types/cfxaddress"
	sdkerrors "github.com/Conflux-Chain/go-conflux-sdk/types/errors"
	"github.com/Conflux-Chain/go-conflux-sdk/utils"
	"github.com/ethereum/go-ethereum/common/hexutil"
	clientRpc "github.com/openweb3/go-rpc-provider"

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
	// space            string = string(types.SPACE_NATIVE)
	receiverListFile string
	weight           decimal.Decimal

	perBatchNum uint
	env         Enviorment
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

func doTransfers(cmd *cobra.Command, args []string) {
	fmt.Println("Initial enviorment")
	env = *NewEnviorment()
	env.pState.refreshChainIdAndSave(env.chainID)

	defer clearCacheFile()

	receiverInfos := mustParseReceivers()
	env.pState.refreshReceiversAndSave(receiverInfos)

	// list cfx and ctoken for user select
	tokenSymbol, tokenAddress := selectToken()
	fmt.Printf("Selected token: %v, contract address: %v\n", tokenSymbol, env.AddrDisplay(tokenAddress))

	// transfer
	fmt.Println("===== Start batch transfer =====")
	batchSummary := BatchSummary{}

	isContinueBreakPoint := askIfContinueUncompletedTxs(receiverInfos)
	if !isContinueBreakPoint {
		env.pState.clearSendingsAndSave()
	}

	// firstly send last send pendings
	env.pState.setSelectTokenAndSave(tokenSymbol, tokenAddress)
	receiverInfos = receiverInfos[env.pState.SendingStartIdx:]

	logrus.Debugf("a receiverInfos len:%v\n", len(receiverInfos))
	if len(env.pState.SendingBatchElems) > 0 {
		fmt.Printf("==== There are uncompleted tx in last time, start send one batch from %v and length %v ===\n",
			env.pState.SendingStartIdx, len(env.pState.SendingBatchElems))

		sentNum := sendOneBatch(env.pState.SendingBatchElems, &batchSummary)
		receiverInfos = receiverInfos[sentNum:]
		logrus.Debugf("b receiverInfos len:%v\n", len(receiverInfos))
	}

	if len(receiverInfos) > 0 {
		exitIfHasPendingTxs()
		// check balance
		fmt.Println("===== Check if balance enough =====")
		checkBalance(env.client, *env.GetFromAddressOfSpace(), receiverInfos, tokenAddress, tokenSymbol)

		estimates := estimateGasAndCollateral(tokenAddress)
		for len(receiverInfos) > 0 {
			batchNum := int(math.Min(float64(perBatchNum), float64(len(receiverInfos))))

			fmt.Printf("\n===== Start send one batch with %v tx=====\n", batchNum)
			elems := creatOneBatchElems(receiverInfos[:batchNum], tokenAddress, tokenSymbol, estimates)
			sentNum := sendOneBatch(elems, &batchSummary)
			receiverInfos = receiverInfos[sentNum:]
			logrus.Debugf("c receiverInfos len:%v\n", len(receiverInfos))
		}
	}

	fmt.Printf("\n===== All transfer done =====\n")
	fmt.Printf("Summares:\n%v\n", &batchSummary)
	fmt.Printf("===== Complete! =====\n")
}

func sendOneBatch(elems []clientRpc.BatchElem, summay *BatchSummary) int {
	oneBatchSummary := batchSendAndWaitReceipt(elems)
	env.pState.setSendingsAndSave(env.pState.SendingStartIdx+len(elems), nil)

	summay.Merge(oneBatchSummary)

	fmt.Printf("===== Batch sent and executed %v tx done, failed %v =====\n", oneBatchSummary.total, oneBatchSummary.GetTotalFailCount())
	if len(oneBatchSummary.failInfos) > 0 {
		fmt.Printf("Fails details:\n%+v\n", strings.Join(oneBatchSummary.failInfos, "\n"))
	}
	return len(elems)
}

// TODO: bulk estimate for getting exact result
func estimateGasAndCollateral(tokenAddress *cfxaddress.Address) types.Estimate {
	if tokenAddress == nil {
		return types.Estimate{
			GasLimit:              defaultGasLimit,
			GasUsed:               defaultGasLimit,
			StorageCollateralized: types.NewBigInt(0),
		}
	}
	randomAddr := cfxaddress.MustNewFromHex("0x0000000000000000000000000000000100000001")
	data := getTransferData(*tokenAddress, randomAddr, big.NewInt(0)).String()
	callReq := types.CallRequest{
		From: env.GetFromAddressOfSpace(),
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
	logrus.Debugf("estimate <from %v, to %v, data %v> result: gas %v, collateral %v", env.AddrDisplay(callReq.From), env.AddrDisplay(callReq.To), *callReq.Data, em.GasLimit, em.StorageCollateralized)
	return em
}

func creatOneBatchElems(oneBatchReceiver []Receiver, tokenAddress *cfxaddress.Address, tokenSymbol string, estimates types.Estimate) (elems []clientRpc.BatchElem) {
	// env.lastPoint is start with -1, so env.lastPoint+1 is actual index, env.lastPoint + 2 is the index count from 1
	startCnt := env.pState.SendingStartIdx + 1 //- len(oneBatchReceiver)

	rpcBatchElems := []clientRpc.BatchElem{}
	for i, v := range oneBatchReceiver {
		tx := createTx(*env.GetFromAddressOfSpace(), v, tokenAddress, env.nonce, estimates)
		rpcBatchElems = append(rpcBatchElems, createBatchElemItem(tx))

		receiver := cfxaddress.MustNew(v.Address, env.networkID)
		fmt.Printf("%v. Sign send %v to %v with value %v nonce %v done\n",
			startCnt+i, tokenSymbol, env.AddrDisplay(&receiver),
			util.DisplayValueWithUnit(calcValue(weight, v.AmountInCfx)), tx.Nonce)
		env.nonce = env.nonce.Add(env.nonce, big.NewInt(1))
	}
	return rpcBatchElems
}

func createBatchElemItem(tx *types.UnsignedTransaction) clientRpc.BatchElem {
	err := env.client.ApplyUnsignedTransactionDefault(tx)
	util.OsExitIfErr(err, "Failed apply unsigned tx %+v", tx)
	tx.From = env.GetFromAddressOfSpace()

	logrus.Debugf("sign tx: %+v", tx)

	// sign
	// encoded, err := env.am.SignTransaction(*tx)
	encoded, err := env.SignTx(tx)
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

	batchSend(rpcBatchElems, nil)
	isTimeout := waitLastReceipt(rpcBatchElems)
	if isTimeout {
		util.OsExit("failed get receipts error in 1 hour")
		// resendFirstPendingTx(rpcBatchElems)
	}

	_, _, summay := batchGetReceipts(rpcBatchElems)
	return summay
}

// if batch elem error or result is empty then set needSends to true
func batchSend(rpcBatchElems []clientRpc.BatchElem, needSends []bool) {
	needSends, needSendIdxs, needSendElems := refreshNeedSends(rpcBatchElems, needSends)
	logrus.Debugf("needSends: %v,needSendIdxs:%v\n", needSends, needSendIdxs)
	if len(needSendIdxs) == 0 {
		return
	}

	// wait response
	hashDoneChan := util.WaitSigAndPrintDot()

	e := env.client.BatchCallRPC(needSendElems)
	hashDoneChan <- nil
	util.OsExitIfErr(e, "Batch send error")
	for i, v := range needSendIdxs {
		rpcBatchElems[v] = needSendElems[i]
	}
	fmt.Println("\n== Received tx hash list")

	env.pState.setSendingsAndSave(env.pState.SendingStartIdx, rpcBatchElems)

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
		rpcBatchElems[i] = improveTxGasPrice(v)
	}
	// re-send
	time.Sleep(time.Second * 2)
	batchSend(rpcBatchElems, needSends)
}

func resendFirstPendingTx(rpcBatchElems []clientRpc.BatchElem) {
	time.Sleep(5 * time.Second)
	// for {
	pendingTxIdx := getFirstPendingTx(rpcBatchElems)
	if pendingTxIdx == -1 {
		return
	}

	fmt.Printf("First pending tx index %v\n", pendingTxIdx)
	reorgnizedTx := improveTxGasPrice(rpcBatchElems[pendingTxIdx])

	rawTx, err := hex.DecodeString(reorgnizedTx.Args[0].(string)[2:])
	util.OsExitIfErr(err, "Failed to decode raw tx string %v", reorgnizedTx.Args[0].(string))

	txHash, err := env.client.SendRawTransaction(rawTx)
	// error must be "tx pool is full" or "nonce tool stale", found "tx already exist"
	if err != nil {
		debug.PrintStack()
		signedTx := &types.SignedTransaction{}
		_ = signedTx.Decode(rawTx, env.networkID)
		logrus.Debugf("Failed to send raw tx %x, docoded %+v, error type %t reflect.Type %v %v", rawTx, signedTx, err, reflect.TypeOf(err), err)

		oldTxHash := types.Hash(*rpcBatchElems[pendingTxIdx].Result.(*string))
		receipt, err := env.client.GetTransactionReceipt(oldTxHash)
		logrus.Debugf("get receipt of old tx hash %v %v", oldTxHash, receipt)
		util.OsExitIfErr(err, "Failed to get receipt of %v", oldTxHash)

		// if receipt null , means "tx pool is full" or "tx already exist",  continue
		// 实际情况中，err 还有 gas price 太低， nonce 太低； nonce 太低的情况不影响；
		// 实际不低，但报 gas price 太低的情况可能是这笔交易已经进到tx pool缓存区，但还没有进tx pool， 而full node取错了交易（暂时看是full node bug）
		// 第一次发 返回 gas price 太低，再发就是tx already exist
		if receipt == nil {
			logrus.Debugf("receipt is nil, tx pool is full or tx already exist")
			// os.Exit(0)
			resendFirstPendingTx(rpcBatchElems)
			return
		}

		// otherwise error must be "nonce tool stale", so re-wait last receipt
		logrus.Debugf("after re-send error and receipt not null, wait receipt")
		if waitLastReceipt(rpcBatchElems) {
			logrus.Debugf("wait receipt timeout1")
			resendFirstPendingTx(rpcBatchElems)
			return
		}

		return
	}

	// refresh tx hash of first pending tx and re-wait last receipt
	txHashStr := txHash.String()
	rpcBatchElems[pendingTxIdx] = reorgnizedTx
	rpcBatchElems[pendingTxIdx].Result = &txHashStr

	env.pState.setSendingsAndSave(env.pState.SendingStartIdx, rpcBatchElems)

	logrus.Debugf("after re-fresh txhash %v, wait receipt", txHashStr)
	if waitLastReceipt(rpcBatchElems) {
		logrus.Debugf("wait receipt timeout2")
		resendFirstPendingTx(rpcBatchElems)
		return
	}

	// }
}

func refreshNeedSends(rpcBatchElems []clientRpc.BatchElem, needSends []bool) (populated []bool, needSendIdx []int,
	needsendElems []clientRpc.BatchElem) {
	// No result or error is not nil, need send
	if needSends == nil {
		needSends = make([]bool, len(rpcBatchElems))
		for i := range needSends {
			result := rpcBatchElems[i].Result.(*string)
			needSends[i] = rpcBatchElems[i].Error != nil || *result == ""
		}
	}

	if len(rpcBatchElems) != len(needSends) {
		util.OsExit("batch elem length must equal to isSend flgas length")
	}

	for i := range needSends {
		if needSends[i] {
			needSendIdx = append(needSendIdx, i)
			needsendElems = append(needsendElems, rpcBatchElems[i])
		}
	}

	// if no need send, return
	return needSends, needSendIdx, needsendElems
}

func improveTxGasPrice(be clientRpc.BatchElem) clientRpc.BatchElem {
	rawTxStr := be.Args[0].(string)
	rawTxByts, err := hex.DecodeString(rawTxStr[2:])
	util.OsExitIfErr(err, "Failed to decode raw tx string %v", rawTxStr)

	signedTx := &types.SignedTransaction{}
	err = signedTx.Decode(rawTxByts, env.networkID)
	util.OsExitIfErr(err, "Failed to decode signed tx %v", signedTx)

	logrus.Debugf("before gasPrice %v\n", signedTx.UnsignedTransaction.GasPrice.ToInt())
	signedTx.UnsignedTransaction.GasPrice = types.NewBigIntByRaw(new(big.Int).Add(signedTx.UnsignedTransaction.GasPrice.ToInt(), big.NewInt(1000)))
	reorgedItem := createBatchElemItem(&signedTx.UnsignedTransaction)
	logrus.Debugf("after gasPrice %v\n", signedTx.UnsignedTransaction.GasPrice.ToInt())

	fmt.Printf("Reorgnized tx with new gas price %v\n", signedTx.UnsignedTransaction.GasPrice.ToInt())
	logrus.Debugf("Reorgnized tx %+v with new gas price\n", signedTx)
	return reorgedItem
}

func getFirstPendingTx(rpcBatchElems []clientRpc.BatchElem) (index int) {
	receipts, _, _ := batchGetReceipts(rpcBatchElems)
	logrus.Debugf("receipts %v\n", receipts)
	// for i := range receipts {
	// 	logrus.Debugf("receipts[i] %v\n", *receipts[i])
	// }

	for i, v := range receipts {
		if reflect.DeepEqual(*v, types.TransactionReceipt{}) {
			return i
		}
	}
	// os.Exit(0)
	return -1
}

func batchGetReceipts(rpcBatchElems []clientRpc.BatchElem) ([]*types.TransactionReceipt, []*error, BatchSummary) {
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

	summary := BatchSummary{
		total: len(rpcBatchElems),
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
		if reflect.DeepEqual(*r, types.TransactionReceipt{}) {
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
	return txReceipts, receiptErrors, summary
}

func waitLastReceipt(rpcBatchElems []clientRpc.BatchElem) (timeout bool) {
	// wait last be packed
	var lastHash *types.Hash
	for i := len(rpcBatchElems); i > 0; i-- {
		if rpcBatchElems[i-1].Error == nil {
			lastHash = (*types.Hash)(rpcBatchElems[i-1].Result.(*string))
			break
		}
	}
	// TODO: impossible occur now;
	// all are error, return
	if lastHash == nil {
		fmt.Println("Failed to send all of this batch of transactions")
		return
	}

	fmt.Printf("\nBatch sent %v, wait last valid tx hash be executed: %v ", len(rpcBatchElems), lastHash)

	receiptDoneChan := util.WaitSigAndPrintDot()
	_, e := env.client.WaitForTransationReceipt(*lastHash, time.Second)
	receiptDoneChan <- nil

	if e == sdkerrors.ErrTimeout {
		return true
	}

	if e != nil {
		util.OsExitIfErr(e, "Failed to get receipt of %+v", lastHash)
	}

	fmt.Printf(" executed! \n\n")
	return false
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
	// data, err := ctoken.GetData("send", reciever.MustGetCommonAddress(), amountInDrip, []byte{})
	data, err := ctoken.GetData("transfer", reciever.MustGetCommonAddress(), amountInDrip)
	util.OsExitIfErr(err, "Failed to get data of transfer ctoken %v to %v amount %v", env.AddrDisplay(&contractAddress), env.AddrDisplay(&reciever), amountInDrip)
	return data
}

func calcValue(numberPerTime decimal.Decimal, weigh decimal.Decimal) *big.Int {
	return weight.Mul(weigh).Mul(decimal.NewFromInt(1e18)).BigInt()
}

// ================= user interact ========================
func selectToken() (symbol string, contractAddress *types.Address) {
	if env.isDebugMode {
		return "", nil
	}

	url := env.GetTokenListUrl()
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
	fmt.Println("\nThese are the token list you could batch transfer:")
	fmt.Printf("%v. Token: %v\n", 1, "CFX")
	for i := range tokenList.List {
		fmt.Printf("%v. Token: %v, Contract Address: %v\n", i+2, tokenList.List[i].Symbol, env.AddrDisplay(&tokenList.List[i].Address))
	}

	selectedIdx := getSelectedIndex(len(tokenList.List) + 2)
	if selectedIdx == 1 {
		symbol = "CFX"
		return
	}
	token := tokenList.List[selectedIdx-2]
	// if token.Symbol != "FC" && token.Symbol[0:1] != "c" {
	// 	util.OsExit("Not support %v currently, please select token FC or starts with 'c', such as cUsdt, cMoon and so on.", token.Symbol)
	// }
	return token.Symbol, &token.Address
}

func askIfContinueUncompletedTxs(receivers []Receiver) bool {
	if env.isDebugMode {
		return true
	}

	// maybe SendingStartIdx is 0 and env.pState.SendingBatchElems not empty, that means them are sent but not received receipt yet.
	hasUncompletedTx := (env.pState.SendingStartIdx + len(env.pState.SendingBatchElems)) > 0

	// if SendingStartIdx is 0 and SendingBatchElems not empty, the uncompletedTxCount is all records.
	uncompletedTxCount := len(receivers) - (env.pState.SendingStartIdx)
	if !hasUncompletedTx {
		return false
	}

	fmt.Printf("There are still %v transactions that were not completed sent last time, you can check detail in transfer_result.json file, 'Y' to continue to sent uncompleted transactions, 'N' to start send form begin\n", uncompletedTxCount)
	var isContinue string
	for {
		fmt.Scanln(&isContinue)
		if isContinue == "Y" || isContinue == "y" {
			return true
		}
		if isContinue == "N" || isContinue == "n" {
			return false
		}
		fmt.Printf("Input must be 'Y' or 'N', please input again\n")
	}
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

func formatReceiverNumber(sentWeightInStr string) {
	var err error
	weight, err = decimal.NewFromString(sentWeightInStr)
	util.OsExitIfErr(err, "wight %v is not a number", sentWeightInStr)
}

// SetParent sets parent command
func SetParent(parent *cobra.Command) {
	parent.AddCommand(rootCmd)
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

	_balance, err := client.GetBalance(*env.GetFromAddressOfSpace())
	cfxBalance = _balance.ToInt()
	util.OsExitIfErr(err, "Failed to get CFX balance of %v", env.AddrDisplay(&from))

	if token != nil {
		contract := common.MustGetCTokenContract(token.String())
		err := contract.Call(nil, &tokenBalance, "balanceOf", from.MustGetCommonAddress())
		util.OsExitIfErr(err, "Failed to get token %v balance of %v", tokenSymbol, env.AddrDisplay(&from))

		_price := (*hexutil.Big)(account.MustParsePrice())
		em := estimateGasAndCollateral(token)

		perTxGasNeed = new(big.Int).Mul(account.MustParsePrice(), em.GasLimit.ToInt())
		if env.space == types.SPACE_NATIVE {
			aginstResp, err := client.CheckBalanceAgainstTransaction(from, *token, em.GasLimit, _price, em.StorageCollateralized)
			util.OsExitIfErr(err, "Failed to check balance against tx")

			logrus.Debugf("CheckBalanceAgainstTransaction from %v gaslimit %v gasprice %v storage collateral %v : %+v", env.AddrDisplay(&from), em.GasLimit, _price, em.StorageCollateralized, aginstResp)

			// needPayGas = aginstResp.WillPayTxFee
			// needPayStorage = aginstResp.WillPayCollateral
			if aginstResp.WillPayTxFee {
				perTxGasNeed = new(big.Int).Mul(account.MustParsePrice(), em.GasLimit.ToInt())
			}
			if aginstResp.WillPayCollateral {
				perUintNeed := new(big.Int).Div(big.NewInt(1e18), big.NewInt(1024))
				perTxStorageNeed = new(big.Int).Mul(perUintNeed, em.StorageCollateralized.ToInt())
			}
		}
	}

	for _, v := range receivers {
		aReceiveNeed := calcValue(weight, v.AmountInCfx)
		// gasFee := big.NewInt(0)
		// if token == nil {
		// 	gasFee = big.NewInt(1).Mul(defaultGasLimit.ToInt(), account.MustParsePrice())
		// }

		receiveNeed = receiveNeed.Add(receiveNeed, aReceiveNeed)
		gasNeed = gasNeed.Add(gasNeed, perTxGasNeed)
		storageNeed = storageNeed.Add(storageNeed, perTxStorageNeed)
	}

	if token == nil {
		cfxNeed := big.NewInt(0).Add(receiveNeed, gasNeed)
		cfxNeed = big.NewInt(0).Add(cfxNeed, storageNeed)
		if cfxBalance.Cmp(cfxNeed) < 0 {
			// clearCacheFile()
			msg := fmt.Sprintf("Balance of %v is not enough,  need %v, has %v",
				env.AddrDisplay(&from), util.DisplayValueWithUnit(cfxNeed), util.DisplayValueWithUnit(cfxBalance))
			util.OsExit(msg)
		}
	} else {
		cfxNeed := big.NewInt(0).Add(gasNeed, storageNeed)
		if cfxBalance.Cmp(cfxNeed) < 0 || tokenBalance.Cmp(receiveNeed) < 0 {
			// clearCacheFile()
			msg := fmt.Sprintf("Token %v balance of %v is not enough or CFX balance not enough to pay gas, "+
				"%v need %v, has %v, CFX need %v, has %v",
				tokenSymbol, env.AddrDisplay(&from),
				tokenSymbol, util.DisplayValueWithUnit(receiveNeed, tokenSymbol), util.DisplayValueWithUnit(tokenBalance, tokenSymbol),
				util.DisplayValueWithUnit(cfxNeed), util.DisplayValueWithUnit(cfxBalance),
			)
			util.OsExit(msg)
		}
	}

	if token == nil {
		tokenBalance = cfxBalance
	}

	fmt.Printf("Balance of %v is enough, %v need %v, fee need %v; token has %v, cfx has %v\n", env.AddrDisplay(&from), tokenSymbol,
		util.DisplayValueWithUnit(receiveNeed, tokenSymbol), util.DisplayValueWithUnit(new(big.Int).Add(gasNeed, storageNeed)),
		util.DisplayValueWithUnit(tokenBalance, tokenSymbol), util.DisplayValueWithUnit(cfxBalance))
}

func exitIfHasPendingTxs() {
	// exit if has pending tx
	from := env.GetFromAddressOfSpace()
	nonce, err := env.client.GetNextNonce(*from)
	util.OsExitIfErr(err, "Failed to get account next nonce")

	pendingNonce, err := env.client.TxPool().NextNonce(*from)
	util.OsExitIfErr(err, "Failed to get account pending nonce")
	if nonce.ToInt().Cmp(pendingNonce.ToInt()) < 0 {
		fmt.Printf("Exit, account %v has pending txs, please clear it first\n", env.AddrDisplay(from))
		os.Exit(0)
	}
}
