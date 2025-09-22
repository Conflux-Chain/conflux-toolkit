package transfer

import (
	"math/big"
	"testing"

	"github.com/shopspring/decimal"
	"gotest.tools/assert"
)

func TestCalcAmountInBaseUnits(t *testing.T) {
	weight := decimal.NewFromInt(1)
	amountInDecimals := decimal.NewFromFloat(0.003)
	decimals := int64(6)
	amountInBaseUnits := calcAmountInBaseUnits(weight, amountInDecimals, decimals)
	assert.Equal(t, amountInBaseUnits.Cmp(big.NewInt(3000)), 0)
}
