package transfer

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"

	"github.com/Conflux-Chain/conflux-toolkit/account"
	"github.com/Conflux-Chain/conflux-toolkit/rpc"
	"github.com/Conflux-Chain/conflux-toolkit/util"
	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	"github.com/Conflux-Chain/go-conflux-sdk/middleware"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/Conflux-Chain/go-conflux-sdk/types/cfxaddress"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/sirupsen/logrus"
)

type Enviorment struct {
	client      *sdk.Client
	am          *sdk.AccountManager
	ethKeystore *keystore.KeyStore

	from       types.Address
	fromEspace common.Address

	nonce              *big.Int
	chainID, networkID uint32
	space              types.SpaceType
	epochHeight        uint64
	pState             ProcessState

	isDebugMode bool
	logLevel    logrus.Level
}

func NewEnviorment() *Enviorment {
	env := Enviorment{}
	env.am = account.DefaultAccountManager
	env.ethKeystore = keystore.NewKeyStore("keystore", keystore.StandardScryptN, keystore.StandardScryptP)
	env.setByProcessEnv()

	env.client = rpc.MustCreateClientWithRetry(3)
	env.client.SetAccountManager(env.am)

	if env.logLevel >= logrus.DebugLevel {
		env.client.UseCallRpcMiddleware(middleware.CallRpcConsoleMiddleware)
		env.client.UseBatchCallRpcMiddleware(middleware.BatchCallRpcConsoleMiddleware)
	}

	status, err := env.client.GetStatus()
	util.OsExitIfErr(err, "Failed to get status")

	env.chainID = uint32(status.ChainID)
	env.networkID = uint32(status.NetworkID)
	env.setSpace()

	env.pState = loadProcessState()
	env.from = cfxaddress.MustNew(account.MustParseAccount().GetHexAddress(), env.networkID)
	env.fromEspace = mustGetAccount(env.ethKeystore, env.from.MustGetCommonAddress()).Address
	// env.pState.refreshSpaceAndSave(types.SpaceType(space))
	env.pState.refreshSenderAndSave(&env.from)

	password := inputPassword()
	err = env.am.Unlock(env.from, password)
	util.OsExitIfErr(err, "Failed to unlock account")

	ethAcc := mustGetAccount(env.ethKeystore, env.from.MustGetCommonAddress())
	err = env.ethKeystore.Unlock(ethAcc, password)
	util.OsExitIfErr(err, "Failed to unlock account")

	fmt.Printf("Account %v is unlocked\n", env.AddrDisplay(&env.from))

	// get inital Nonce
	_nonce, e := env.client.TxPool().NextNonce(*env.GetFromAddressOfSpace())
	env.nonce = _nonce.ToInt()
	util.OsExitIfErr(e, "Failed to get nonce of from %v", env.AddrDisplay(&env.from))

	epoch, err := env.client.GetEpochNumber(types.EpochLatestState)
	util.OsExitIfErr(err, "Failed to get epoch")
	env.epochHeight = epoch.ToInt().Uint64()
	return &env
}

func (env *Enviorment) AddrDisplay(addr *cfxaddress.Address) string {
	if addr == nil {
		return "nil"
	}

	switch env.space {
	case types.SPACE_NATIVE:
		return addr.String()
	case types.SPACE_EVM:
		if addr.String() == env.from.String() {
			return env.fromEspace.String()
		}
		return addr.MustGetCommonAddress().String()
	}
	panic("unknown space")
}

// GetFromAddressOfSpace returns cfxaddress of env.from accroding to space
func (env *Enviorment) GetFromAddressOfSpace() *cfxaddress.Address {
	switch env.space {
	case types.SPACE_NATIVE:
		return &env.from
	case types.SPACE_EVM:
		addr := cfxaddress.MustNewFromCommon(env.fromEspace, env.networkID)
		return &addr
	}
	panic("unknown space")
}

func (env *Enviorment) GetTokenListUrl() string {
	switch env.networkID {
	case util.MAINNET:
		return "https://api.confluxscan.net/account/tokens?account="
	case util.TESTNET:
		return "https://api-testnet.confluxscan.net/account/tokens?account="
	case util.ESPACE_MAINNET:
		return "https://evmapi.confluxscan.net/account/tokens?account="
	case util.ESPACE_TESTNET:
		return "https://evmapi-testnet.confluxscan.net/account/tokens?account="
	}
	panic("unknown network")
}

func (env *Enviorment) SignTx(tx *types.UnsignedTransaction) ([]byte, error) {
	switch env.space {
	case types.SPACE_NATIVE:
		return env.am.SignTransaction(*tx)
	case types.SPACE_EVM:
		eTx, addr, chainID := cfxToEthTx(tx)
		eTx = signEthLegacyTx(env.ethKeystore, addr, eTx, chainID)
		return rlp.EncodeToBytes(eTx)
	}
	return nil, errors.New("unkown space")
}

func (env *Enviorment) DecodeTx(rawTxBytes []byte) *types.UnsignedTransaction {
	switch env.space {
	case types.SPACE_NATIVE:
		unsignedTx := &types.UnsignedTransaction{}
		err := unsignedTx.Decode(rawTxBytes, env.networkID)
		util.OsExitIfErr(err, "Failed to decode signed tx %v to cfx tx", rawTxBytes)
		return unsignedTx
	case types.SPACE_EVM:
		etx := ethtypes.Transaction{}
		err := rlp.DecodeBytes(rawTxBytes, &etx)
		util.OsExitIfErr(err, "Failed to decode signed tx %v to eth tx", rawTxBytes)
		return ethToCfxTx(&etx, env.chainID)
	}
	panic("unknown space")
}

func (env *Enviorment) setSpace() {
	// var space types.SpaceType
	switch env.networkID {
	case util.MAINNET:
		env.space = types.SPACE_NATIVE
		return
	case util.TESTNET:
		env.space = types.SPACE_NATIVE
		return
	case util.ESPACE_MAINNET:
		env.space = types.SPACE_EVM
		return
	case util.ESPACE_TESTNET:
		env.space = types.SPACE_EVM
		return
	}
	panic("unknown network")
}

func (env *Enviorment) setByProcessEnv() {
	modeStr := os.Getenv("MODE")
	if modeStr == "DEBUG" || modeStr == "debug" {
		env.isDebugMode = true
	}

	env.logLevel = logrus.InfoLevel
	logLevelStr := os.Getenv("LOG_LEVEL")
	if logLevelStr != "" {
		logLevel, err := strconv.Atoi(logLevelStr)
		if err != nil {
			util.OsExitIfErr(err, "Failed to get log level")
		}
		env.logLevel = logrus.Level(logLevel)
	}
}

func inputPassword() string {
	if env.isDebugMode {
		return "123"
	}
	return account.MustInputPassword("Enter password: ")
}
