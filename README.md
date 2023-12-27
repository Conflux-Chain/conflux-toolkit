# Conflux Toolkit

Conflux toolkit is a CLI tool to interact with full node and manage accounts at local file system.

## Build
Run `go build` under the root directory to generate binary.

## Subcommands
Run `./conflux-toolkit` to view all supported subcommands, including:

- `account`: account management.
- `rpc`: interact with full node.
- `contract`: interact with smart contract.
- `transfer`: batch transfer cfx to receivers.

For more details, run with `-h` flag.

## BulkSender tool

## Build

`make build` will create `mac` and `windows` in `./dist` folder, there are the workspace for execute BulkSender tool.

## Run

Use `./dist/mac` on Mac OS and `./dist/windows` on Windows OS.

1. Fill receiver list on `1_填写空投列表.csv`; the 1st column means "Receiver address" and 2nd column means "Send amount in CFX"

2. Run `2_双击开启空投_mac.command` on windows OS or run `2_双击开启空投_windows.`sh` on Mac OS