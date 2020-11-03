package account

import (
	"fmt"
	"os"
	"strconv"

	"github.com/howeyc/gopass"
)

// Account is managed account under keystore
var Account string

// MustParseAccount parse account from input parameter.
func MustParseAccount() string {
	accountIndex, err := strconv.Atoi(Account)
	if err != nil {
		return Account
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
