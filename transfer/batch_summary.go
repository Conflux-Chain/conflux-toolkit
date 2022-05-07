package transfer

import (
	"fmt"
	"strings"
)

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
