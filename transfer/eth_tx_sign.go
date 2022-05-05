package transfer

import (
	"math/big"

	cfxtypes "github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/Conflux-Chain/go-conflux-sdk/utils/addressutil"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/sirupsen/logrus"
)

func signEthLegacyTx(ks *keystore.KeyStore, from common.Address, tx *types.Transaction, chainID *big.Int) *types.Transaction {
	acc := mustGetAccount(ks, from)
	signedTx, err := ks.SignTx(acc, tx, chainID)
	if err != nil {
		panic(err)
	}
	return signedTx
}

func adaptToEthTx(tx *cfxtypes.UnsignedTransaction) (*types.Transaction, common.Address, *big.Int) {
	logrus.Info("adaptToEthTx", tx)
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

func mustGetAccount(ks *keystore.KeyStore, addr common.Address) accounts.Account {
	accs := ks.Accounts()
	logrus.WithFields(logrus.Fields{
		"addr": addr,
		"accs": accs,
	}).Info("mustGetAccount")
	for _, acc := range accs {
		source := addressutil.EtherAddressToCfxAddress(acc.Address, false, 1)
		target := addressutil.EtherAddressToCfxAddress(addr, false, 1)
		if source.String() == target.String() {
			return acc
		}
	}
	panic("account not found")
}
