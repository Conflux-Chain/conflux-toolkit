
test:
	echo "hello"

build_mac:
	go build .
	mv ./conflux-toolkit ./dist/mac
	cp -r ./scripts/batch_transfer/* ./dist/mac/
	cd ./dist/mac && zip -r ./batch-transfer-mac.zip ./* -x *.zip *.txt && cd ../../

build_windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build
	mv ./conflux-toolkit.exe ./dist/windows/conflux-toolkit
	cp -r ./scripts/batch_transfer/* ./dist/windows/
	cd ./dist/windows && zip -r -l ./batch-transfer-windows.zip ./* -x *.zip *.txt && cd ../..

build:
	make build_mac
	make build_windows