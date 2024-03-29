package account

import (
	"errors"
	"fmt"
	"math/big"

	cfxtypes "github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/Conflux-Chain/go-conflux-sdk/types/cfxaddress"
	"github.com/Conflux-Chain/go-conflux-sdk/utils/addressutil"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var ErrInvalidSig = errors.New("invalid transaction v, r, s values")

func SignEthLegacyTx(ks *keystore.KeyStore, from common.Address, tx *types.Transaction, chainID *big.Int) *types.Transaction {
	acc := MustGetAccount(ks, from)
	signedTx, err := ks.SignTx(acc, tx, chainID)
	if err != nil {
		panic(err)
	}
	return signedTx
}

func SignEthLegacyTxWithPasswd(ks *keystore.KeyStore, from common.Address, tx *types.Transaction, chainID *big.Int, passwd string) *types.Transaction {
	acc := MustGetAccount(ks, from)
	fmt.Printf("Espace Address %v\n", acc.Address)
	signedTx, err := ks.SignTxWithPassphrase(acc, passwd, tx, chainID)
	if err != nil {
		panic(err)
	}
	return signedTx
}

func recoverPlain(sighash common.Hash, R, S, Vb *big.Int, homestead bool) (common.Address, error) {
	if Vb.BitLen() > 8 {
		return common.Address{}, ErrInvalidSig
	}
	V := byte(Vb.Uint64() - 27)
	if !crypto.ValidateSignatureValues(V, R, S, homestead) {
		return common.Address{}, ErrInvalidSig
	}
	// encode the signature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, crypto.SignatureLength)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V
	// recover the public key from the signature
	pub, err := crypto.Ecrecover(sighash[:], sig)
	if err != nil {
		return common.Address{}, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return common.Address{}, errors.New("invalid public key")
	}
	var addr common.Address
	copy(addr[:], crypto.Keccak256(pub[1:])[12:])
	return addr, nil
}

func CfxToEthTx(tx *cfxtypes.UnsignedTransaction) (*types.Transaction, common.Address, *big.Int) {
	// logrus.Info("adaptToEthTx", tx)
	from := tx.From.MustGetCommonAddress()
	nonce := tx.Nonce.ToInt().Uint64()
	to := tx.To.MustGetCommonAddress()
	amount := tx.Value.ToInt()
	gasLimit := tx.Gas.ToInt().Uint64()
	gasPrice := tx.GasPrice.ToInt()
	data := tx.Data
	chainID := big.NewInt(int64(*tx.ChainID))

	return types.NewTransaction(nonce, to, amount, gasLimit, gasPrice, data), from, chainID
}

func EthToCfxTx(etx *types.Transaction, chainID uint32) *cfxtypes.UnsignedTransaction {
	utx := &cfxtypes.UnsignedTransaction{}
	// utx.ChainID = cfxtypes.NewBigIntByRaw(chainID).ToInt().Uint64()
	// utx.From = etx.From()
	to := cfxaddress.MustNewFromCommon(*etx.To(), chainID)
	chainid := hexutil.Uint(uint(chainID))

	utx.Nonce = cfxtypes.NewBigInt(etx.Nonce())
	utx.To = &to
	utx.Value = cfxtypes.NewBigIntByRaw(etx.Value())
	utx.Gas = cfxtypes.NewBigInt(etx.Gas())
	utx.GasPrice = cfxtypes.NewBigIntByRaw(etx.GasPrice())
	utx.Data = etx.Data()
	utx.ChainID = &chainid

	return utx
}

func MustGetAccount(ks *keystore.KeyStore, addr common.Address) accounts.Account {
	accs := ks.Accounts()

	for _, acc := range accs {
		source := addressutil.EtherAddressToCfxAddress(acc.Address, false, 1)
		target := addressutil.EtherAddressToCfxAddress(addr, false, 1)
		if source.String() == target.String() {
			return acc
		}
	}
	panic("account not found")
}
