package account

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

func TestRecoverPlain(t *testing.T) {
	h := common.HexToHash("0x136d63819ab69462c3faaa7ffe18253b6277bce3863a6de382875ab4b7f84e50")
	r, _ := hexutil.DecodeBig("0xfbb4754c7d260b5c62d9fbb914dd9153aa4f0ba1daa6018a8cb1d42bd962d101")
	s, _ := hexutil.DecodeBig("0x73312bb39f46316b91b1c41a843cc86a8cb5fd30d4ae7dfa699b9c9e46901cfb")
	v := big.NewInt(27) // hexutil.DecodeBig("0xb2")

	addr, err := recoverPlain(h, r, s, v, false)
	assert.NoError(t, err)

	fmt.Printf("addr %v\n", addr)
}

func TestSignH(t *testing.T) {
	h := common.Hash{}
	prv, _ := crypto.HexToECDSA("9ec393923a14eeb557600010ea05d635c667a6995418f8a8f4bdecc63dfe0bb9")
	sig, _ := crypto.Sign(h[:], prv)
	addr := crypto.PubkeyToAddress(prv.PublicKey)
	fmt.Printf("addr %v sig %x\n", addr, sig)

	r, s, v := sig[0:32], sig[32:64], sig[64]
	fmt.Printf("r %x s %x v %x\n", r, s, v)

	var rAddr common.Address
	pub, err := crypto.Ecrecover(h[:], sig)
	assert.NoError(t, err)
	copy(rAddr[:], crypto.Keccak256(pub[1:])[12:])
	fmt.Printf("recovered addr %v\n", rAddr)
}

func TestTxHash(t *testing.T) {
	
}
