package transfer

import (
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/shopspring/decimal"
)

// Receiver represents transfer receiver infomation
type Receiver struct {
	Address types.Address
	Weight  decimal.Decimal
}
