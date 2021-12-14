module github.com/Conflux-Chain/conflux-toolkit

go 1.14

require (
	github.com/Conflux-Chain/go-conflux-sdk v1.0.15
	github.com/emirpasic/gods v1.12.0
	github.com/ethereum/go-ethereum v1.9.25
	github.com/howeyc/gopass v0.0.0-20190910152052-7cb4b85ec19c
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/pkg/errors v0.9.1
	github.com/shopspring/decimal v1.2.0
	github.com/sirupsen/logrus v1.2.0
	github.com/spf13/cobra v1.0.0
	github.com/stretchr/testify v1.5.1 // indirect
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad // indirect
	golang.org/x/text v0.3.5 // indirect
)

// replace github.com/Conflux-Chain/go-conflux-sdk v1.0.15 => github.com/wangdayong228/go-conflux-sdk v0.2.0
replace github.com/Conflux-Chain/go-conflux-sdk v1.0.15 => ../go-conflux-sdk
