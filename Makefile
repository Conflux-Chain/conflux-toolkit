
test:
	echo "hello"

build_mac:
	go build .
	mkdir -p ./dist
	mv ./conflux-toolkit ./dist/mac
	cp -r ./scripts/batch_transfer/* ./dist/mac/

build_windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build
	mkdir -p ./dist
	mv ./conflux-toolkit.exe ./dist/windows/conflux-toolkit
	cp -r ./scripts/batch_transfer/* ./dist/windows/

build:
	make build_mac
	make build_windows

zip:
	make build
	cd ./dist/mac && zip -r ./batch-transfer-mac.zip ./* -x *.zip *.txt && cd ../../
	cd ./dist/windows && zip -r -l ./batch-transfer-windows.zip ./* -x *.zip *.txt && cd ../..
