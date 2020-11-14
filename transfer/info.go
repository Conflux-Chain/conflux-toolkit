package transfer

import (
	"github.com/Conflux-Chain/go-conflux-sdk/types"
)

// Receiver represents transfer receiver infomation
type Receiver struct {
	Address types.Address
	Weight  uint
}
