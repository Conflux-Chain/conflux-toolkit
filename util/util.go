package util

import (
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
)

const (
	MAINNET = 1029
	TESTNET = 1
)

// CreateUsageCommand creates a command to display help.
func CreateUsageCommand(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
}

// MustParseBigInt parses the specified value to number and returns value * 10 ^ exp.
func MustParseBigInt(value string, exp int32) *big.Int {
	num, err := decimal.NewFromString(value)
	if err != nil {
		fmt.Println("invalid decimal format for value:", err.Error())
		os.Exit(1)
	}

	return decimal.NewFromBigInt(big.NewInt(1), exp).Mul(num).BigInt()
}

// DisplayValueWithUnit returns the display format for given drip value.
func DisplayValueWithUnit(drip *big.Int, tokenSymbol ...string) string {
	if big.NewInt(1_000).Cmp(drip) > 0 {
		return fmt.Sprintf("%v Drip", drip)
	}

	if big.NewInt(1_000_000).Cmp(drip) > 0 {
		return fmt.Sprintf("%v Kdrip", decimal.NewFromBigInt(drip, -3))
	}

	if big.NewInt(1_000_000_000).Cmp(drip) > 0 {
		return fmt.Sprintf("%v Mdrip", decimal.NewFromBigInt(drip, -6))
	}

	if big.NewInt(1_000_000_000_000).Cmp(drip) > 0 {
		return fmt.Sprintf("%v Gdrip", decimal.NewFromBigInt(drip, -9))
	}

	_tokenSymbol := "CFX"
	if len(tokenSymbol) > 0 && tokenSymbol[0] != "" {
		_tokenSymbol = tokenSymbol[0]
	}

	return fmt.Sprintf("%v %v", decimal.NewFromBigInt(drip, -18), _tokenSymbol)
}

// OsExitIfErr prints error msg and exit
func OsExitIfErr(err error, format string, a ...interface{}) {
	if err != nil {
		fmt.Printf("\nError: %+v\n", errors.Wrapf(err, format, a...))
		os.Exit(1)
	}
}

func PanicIfErr(err error, format string, a ...interface{}) {
	if err != nil {
		errMsg := fmt.Sprintf(format, a...)
		errMsg += fmt.Sprintf("\nError: %v", err.Error())
		panic(errMsg)
	}
}

// OsExit prints msg and exit
func OsExit(format string, a ...interface{}) {
	fmt.Printf("\nError: %+v\n", errors.Errorf(format, a...))
	os.Exit(1)
}

func Panic(format string, a ...interface{}) {
	errMsg := fmt.Sprintf(format, a...)
	panic(errMsg)
}

func WaitSigAndPrintDot() chan interface{} {
	doneChan := make(chan interface{})
	go func() {
		for {
			time.Sleep(time.Second)
			select {
			case <-doneChan:
				return
			default:
				fmt.Printf(".")
			}
		}
	}()
	return doneChan
}

func GetStringVal(strPtr *string) string {
	if strPtr == nil {
		return "nil"
	}
	return *strPtr
}
