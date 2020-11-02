package account

import (
	"fmt"
	"os"

	"github.com/howeyc/gopass"
)

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
