package account

import (
	"fmt"
	"math/big"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/howeyc/gopass"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
)

var (
	account  string
	priceStr string

	// ValueStr is the string representation of value.
	ValueStr string
)

// AddAccountVar adds account variable for specified command.
func AddAccountVar(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&account, "account", "", "Account address in HEX format or address index number")
	cmd.MarkPersistentFlagRequired("account")
}

func addFromVar(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&account, "from", "", "From address in HEX format or address index number")
	cmd.MarkPersistentFlagRequired("from")
}

// AddGasPriceVar addds price variable for specified command.
func AddGasPriceVar(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&priceStr, "price", "1", "Gas price in drip")
}

// MustParseAccount parse account from input parameter.
func MustParseAccount() string {
	accountIndex, err := strconv.Atoi(account)
	if err != nil {
		return strings.ToLower(account)
	}

	accounts := listAccountsAsc()
	if len(accounts) == 0 {
		fmt.Println("No account found!")
		os.Exit(1)
	}

	if accountIndex >= len(accounts) {
		fmt.Println("Invalid account index, it should be between 0 and", len(accounts)-1)
		os.Exit(1)
	}

	fmt.Println("Account:", accounts[accountIndex])

	return accounts[accountIndex]
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
	value, err := decimal.NewFromString(ValueStr)
	if err != nil {
		fmt.Println("invalid decimal format for value:", err.Error())
		os.Exit(1)
	}

	return decimal.NewFromBigInt(big.NewInt(10), 18).Mul(value).BigInt()
}

func listAccountsAsc() []string {
	var accounts []string

	for _, addr := range am.List() {
		accounts = append(accounts, addr.String())
	}

	sort.Strings(accounts)

	return accounts
}

func mustInputAndConfirmPassword() string {
	fmt.Println("Please input password to create key file!")

	passwd1 := mustInputPassword("Enter password: ")
	passwd2 := mustInputPassword("Confirm password: ")

	if passwd1 != passwd2 {
		fmt.Println("Password mismatch!")
		os.Exit(1)
	}

	return passwd1
}

func mustInputPassword(prompt string) string {
	fmt.Print(prompt)

	passwd, err := gopass.GetPasswd()
	if err != nil {
		fmt.Println("Failed to get password:", err.Error())
		os.Exit(1)
	}

	return string(passwd)
}
