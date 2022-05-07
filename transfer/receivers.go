package transfer

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/Conflux-Chain/conflux-toolkit/util"
	"github.com/Conflux-Chain/go-conflux-sdk/types/cfxaddress"
	"github.com/shopspring/decimal"
)

func mustParseReceivers() []Receiver {
	// read csv file
	content, err := ioutil.ReadFile(receiverListFile)
	util.OsExitIfErr(err, "Failed to read file %v", receiverListFile)

	// parse to struct
	lines := strings.Split(string(content), "\n")
	receiverInfos := []Receiver{}

	invalids := []string{}

	for i, v := range lines {
		v = strings.Replace(v, "\t", " ", -1)
		v = strings.Replace(v, ",", " ", -1)
		items := strings.Fields(v)
		if len(v) == 0 {
			continue
		}

		if len(items) != 2 {
			util.OsExit("Line %v: %#v column number is %v, which shoule be 2\n", i, v, len(items))
		}

		isDecimal, err := regexp.Match(`^\d+\.?\d*$`, []byte(items[1]))
		util.OsExitIfErr(err, "Failed to regex check %v ", items[1])
		if !isDecimal {
			invalids = append(invalids, fmt.Sprintf("Line %v: Number %v is unsupported, only supports pure number format, scientific notation like 1e18 and other representation format are unspported", i+1, items[1]))
			continue
		}

		amountInCfx, err := decimal.NewFromString(items[1])
		if err != nil {
			invalids = append(invalids, fmt.Sprintf("Line %v: Failed to parse %v to int, errmsg:%v", i+1, items[1], err.Error()))
			continue
		}

		_, err = cfxaddress.New(items[0])
		if err != nil {
			invalids = append(invalids, fmt.Sprintf("Line %v: Failed to create cfx address by %v, Errmsg: %v", i+1, items[0], err.Error()))
			continue
		}

		info := Receiver{
			Address:     items[0],
			AmountInCfx: amountInCfx,
		}

		receiverInfos = append(receiverInfos, info)
	}

	if len(invalids) > 0 {
		errMsg := fmt.Sprintf("Invalid Recevier info exists:\n%v", strings.Join(invalids, "\n"))
		util.OsExit(errMsg)
	}

	fmt.Printf("Receiver list count :%+v\n", len(receiverInfos))
	return receiverInfos
}
