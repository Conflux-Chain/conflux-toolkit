package transfer

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"gotest.tools/assert"
)

func Test_decodeTx(t *testing.T) {
	env := Enviorment{
		space:   types.SPACE_EVM,
		chainID: 71,
	}
	rawTxBytes, err := hex.DecodeString("f8640a843b9aca008252089412988210c05a43ebd76575f5421ef84b120ebf82148081b1a05aaa67bb0c06d0df90c17a01eb1a6724f905555590814ac6fe24129e0f46447aa044d0fb5eb10b0b5b6dd0628231fe0fefa59003ecc0c3ebf5ae7036c321b847fd")
	if err != nil {
		t.Fatal(err)
	}
	tx := env.DecodeTx(rawTxBytes)

	expectJson := `{"From":null,"Nonce":"0xa","GasPrice":"0x3b9aca00","Gas":"0x5208","Value":"0x14","StorageLimit":null,"EpochHeight":null,"ChainID":"0x47","To":"net71:aakkvauu2breh481pz49muu89bfvedz9ujaj4mtsjc","Data":"0x"}`
	actualJson, err := json.Marshal(tx)

	assert.Equal(t, expectJson, string(actualJson))
}
