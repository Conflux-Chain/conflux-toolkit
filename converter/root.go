package converter

import (
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	"github.com/Conflux-Chain/conflux-toolkit/rpc"
	"github.com/Conflux-Chain/conflux-toolkit/util"
	"github.com/Conflux-Chain/go-conflux-sdk/types/cfxaddress"

	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "convert",
		Short: "convert subcommand",
		Run:   doConverter,
	}
	addressListFile string

	// base32 or hex, default is base32
	targetFormat string
	verbose      bool
)

func init() {
	rpc.AddURLVar(rootCmd)

	rootCmd.PersistentFlags().StringVar(&addressListFile, "addresses", "", "address list file path")
	rootCmd.MarkPersistentFlagRequired("addresses")

	rootCmd.PersistentFlags().StringVar(&targetFormat, "to", "base32", "convert target format, base32 or hex, the default is base32")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "if format is base32, set verbose to true to get verbose string")
}

// SetParent sets parent command
func SetParent(parent *cobra.Command) {
	parent.AddCommand(rootCmd)
}

func doConverter(cmd *cobra.Command, args []string) {
	// read file
	inputs := readAddressList()
	// convert
	outputs := convert(inputs)
	// write file
	writeResult(inputs, outputs)
}

func readAddressList() []string {
	// read csv file
	_content, err := ioutil.ReadFile(addressListFile)
	util.OsExitIfErr(err, "Failed to read file %v", addressListFile)

	content := string(_content)
	content = strings.TrimSpace(content)
	// parse to struct
	lines := strings.Split(content, "\n")
	return lines
}

func convert(inputAddresses []string) []string {

	outputs := []string{}
	for _, addr := range inputAddresses {
		var output string
		fmt.Printf("convert input:%v\n", addr)
		cfxAddr := cfxaddress.MustNew(addr, cfxaddress.NetowrkTypeMainnetID)

		switch targetFormat {
		case "hex":
			output = cfxAddr.GetHexAddress()
		case "base32":
			if verbose {
				output = cfxAddr.String()
			} else {
				output = cfxAddr.MustGetBase32Address()
			}
		default:
			util.OsExit("Invalid address format %v", targetFormat)
		}

		outputs = append(outputs, output)
	}
	return outputs
}

func writeResult(inputs []string, outputs []string) {
	if len(inputs) != len(outputs) {
		util.OsExit("inputs length %v not match outpus %v", len(inputs), len(outputs))
	}

	result := "inputs,outputs\n"
	for i := 0; i < len(inputs); i++ {
		result += fmt.Sprintf("%v,%v\n", inputs[i], outputs[i])
	}

	resultPath := path.Join(addressListFile, "../converted_result.csv")
	err := ioutil.WriteFile(resultPath, []byte(result), 0777)
	util.OsExitIfErr(err, "Failed to write result")
}
