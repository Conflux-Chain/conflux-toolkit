package transfer

import (
	"math/big"

	cfxtypes "github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/Conflux-Chain/go-conflux-sdk/types/cfxaddress"
	"github.com/Conflux-Chain/go-conflux-sdk/utils/addressutil"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

func signEthLegacyTx(ks *keystore.KeyStore, from common.Address, tx *types.Transaction, chainID *big.Int) *types.Transaction {
	acc := mustGetAccount(ks, from)
	signedTx, err := ks.SignTx(acc, tx, chainID)
	if err != nil {
		panic(err)
	}
	return signedTx
}

func cfxToEthTx(tx *cfxtypes.UnsignedTransaction) (*types.Transaction, common.Address, *big.Int) {
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

func ethToCfxTx(etx *types.Transaction, chainID uint32) *cfxtypes.UnsignedTransaction {
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

func mustGetAccount(ks *keystore.KeyStore, addr common.Address) accounts.Account {
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
