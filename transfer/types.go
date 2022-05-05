package transfer

import (
	"fmt"
	"math/big"
	"strings"

	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/keystore"
)

type enviorment struct {
	client      *sdk.Client
	am          *sdk.AccountManager
	ethKeystore *keystore.KeyStore
	// SendingsStartIdx          int
	from               types.Address
	nonce              *big.Int
	chainID, networkID uint32
	epochHeight        uint64
	pState             ProcessState
	isDebugMode        bool
}

type BatchSummary struct {
	total                 int
	sentTxfailedCount     int
	getReceiptFailedCount int
	receiptNullCount      int
	executFailedCount     int
	failInfos             []string
}

func (f BatchSummary) GetTotalFailCount() int {
	return f.sentTxfailedCount +
		f.getReceiptFailedCount +
		f.receiptNullCount +
		f.executFailedCount
}

func (f *BatchSummary) Merge(other BatchSummary) {
	f.total += other.total
	f.sentTxfailedCount += other.sentTxfailedCount
	f.getReceiptFailedCount += other.getReceiptFailedCount
	f.receiptNullCount += other.receiptNullCount
	f.executFailedCount += other.executFailedCount
	f.failInfos = append(f.failInfos, other.failInfos...)
}

func (f *BatchSummary) String() string {
	result := fmt.Sprintf("- Total: %v\n- Sent failed: %v\n- Get receipt failed: %v\n- Receipt null: %v\n- Execute failed: %v\n",
		f.total, f.sentTxfailedCount, f.getReceiptFailedCount, f.receiptNullCount, f.executFailedCount)
	if len(f.failInfos) > 0 {
		result += fmt.Sprintf("- Fail infos:\n%v", strings.Join(f.failInfos, "\n"))
	}
	return result
}
