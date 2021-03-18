package transfer

import (
	"github.com/shopspring/decimal"
)

// Receiver represents transfer receiver infomation
type Receiver struct {
	Address string
	Weight  decimal.Decimal
}
