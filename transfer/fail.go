package transfer

import (
	"fmt"

	"github.com/Conflux-Chain/conflux-toolkit/util"
)

// ======= Fail Message =======
func failSentTx(i int, err error) string {
	return fmt.Sprintf("The %vth transaction failed, Failed type: %v, Error Info: %+v", i, "send tx", err)
}

func failGetTxReceipt(i int, err error) string {
	return fmt.Sprintf("The %vth transaction failed, Failed type: %v, Error Info: %+v", i, "get tx receipt", err)
}

func failTxReceiptNull(i int) string {
	return fmt.Sprintf("The %vth transaction failed, Failed type: %v", i, "tx receipt null")
}

func failExecuteTx(i int, errMsg *string) string {
	return fmt.Sprintf("The %vth transaction failed, Failed type: %v, Error Info: %+v", i, "tx execute failed", util.GetStringVal(errMsg))
}
