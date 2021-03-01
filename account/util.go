package account

import (
	"fmt"
	"math/big"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Conflux-Chain/conflux-toolkit/util"
	common "github.com/Conflux-Chain/conflux-toolkit/util"
	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/Conflux-Chain/go-conflux-sdk/types/cfxaddress"
	"github.com/howeyc/gopass"
	"github.com/spf13/cobra"
)

var (
	account  string
	priceStr string

	// ValueCfx is the string representation of value in CFX.
	ValueCfx string

	// DefaultAccountManager is the default account manager under keystore folder.
	DefaultAccountManager *sdk.AccountManager = am
)

// AddAccountVar adds account variable for specified command.
func AddAccountVar(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&account, "account", "", "Account address in HEX format or address index number")
	cmd.MarkPersistentFlagRequired("account")
}

func AddFromVar(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&account, "from", "", "From address in HEX format or address index number")
	cmd.MarkPersistentFlagRequired("from")
}

// AddGasPriceVar addds price variable for specified command.
func AddGasPriceVar(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&priceStr, "price", "1", "Gas price in drip")
}

// MustParseAccount parse account from input parameter.
func MustParseAccount() *types.Address {
	accountIndex, err := strconv.Atoi(account)
	if err != nil {
		return MustNewAccount(strings.ToLower(account))
	}

	accounts := listAccountsAsc()
	if len(accounts) == 0 {
		panic("No account found!")
	}

	if accountIndex >= len(accounts) {
		fmt.Println("Invalid account index, it should be between 0 and", len(accounts)-1)
		os.Exit(1)
	}

	fmt.Println("Account:", accounts[accountIndex])

	return &accounts[accountIndex]
}

// MustParsePrice parse gas price from input parameter.
func MustParsePrice() *big.Int {
	price, ok := new(big.Int).SetString(priceStr, 10)
	if !ok {
		fmt.Println("invalid number format for gas price")
		os.Exit(1)
	}

	return price
}

// MustParseValue parse value in CFX from input parameter.
func MustParseValue() *big.Int {
	return common.MustParseBigInt(ValueCfx, 18)
}

func listAccountsAsc() []types.Address {
	var accounts []types.Address

	for _, addr := range am.List() {
		accounts = append(accounts, addr)
	}

	sort.Slice(accounts, func(i, j int) bool {
		return strings.Compare(accounts[i].GetHexAddress(), accounts[j].GetHexAddress()) == -1
	})

	return accounts
}

func mustInputAndConfirmPassword() string {
	fmt.Println("Please input password to create key file!")

	passwd1 := MustInputPassword("Enter password: ")
	passwd2 := MustInputPassword("Confirm password: ")

	if passwd1 != passwd2 {
		fmt.Println("Password mismatch!")
		os.Exit(1)
	}

	return passwd1
}

// MustInputPassword prompt user to input password.
func MustInputPassword(prompt string) string {
	fmt.Print(prompt)

	passwd, err := gopass.GetPasswd()
	if err != nil {
		fmt.Println("Failed to get password:", err.Error())
		os.Exit(1)
	}

	return string(passwd)
}

// MustNewAccount must create conflux address by base32 string or hex40 string.
func MustNewAccount(base32OrHex string, networkID ...uint32) *types.Address {
	hexPattern := `(?i)^0x[a-f0-9]{40}$`
	base32Pattern := `(?i)^(cfx|net).*:\w{42}$`

	if ok, _ := regexp.Match(hexPattern, []byte(base32OrHex)); ok {
		_networkID := uint32(0)
		if len(networkID) > 0 {
			_networkID = networkID[0]
		}
		_account := cfxaddress.MustNewFromHex(base32OrHex, _networkID)
		return &_account
	}

	if ok, _ := regexp.Match(base32Pattern, []byte(base32OrHex)); ok {
		_account := cfxaddress.MustNewFromBase32(base32OrHex)
		return &_account
	}
	util.OsExit("input %v need be base32 string or hex40 string,", base32OrHex, networkID)
	panic(0)
}
