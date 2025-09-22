package util

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDisplayTokenValueInDecimals(t *testing.T) {
	amountInBaseUnits := big.NewInt(3)
	tokenSymbol := "AxCNH"
	decimals := int64(6)
	str := (DisplayTokenValueInDecimals(amountInBaseUnits, tokenSymbol, decimals))
	assert.Equal(t, str, "0.000003 AxCNH")
}
