
test:
	echo "hello"

build_mac_arm:
	go build .
	mkdir -p ./dist/mac_arm
	mv ./conflux-toolkit ./dist/mac_arm/
	cp -r ./scripts/batch_transfer/* ./dist/mac_arm/

build_mac_amd:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build
	mkdir -p ./dist/mac_amd
	mv ./conflux-toolkit ./dist/mac_amd/
	cp -r ./scripts/batch_transfer/* ./dist/mac_amd/

build_windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build
	mkdir -p ./dist/windows
	mv ./conflux-toolkit.exe ./dist/windows/conflux-toolkit
	cp -r ./scripts/batch_transfer/* ./dist/windows/

build:
	make build_mac_arm
	make build_mac_amd
	make build_windows

zip:
	make build
	mkdir -p ./dist/zip
	cd ./dist/mac_amd && zip -r ./batch-transfer-mac.zip ./* -x *.zip -x *.txt -x "keystore/*" && mv batch-transfer-mac.zip ../zip/ && cd ../../
	cd ./dist/windows && zip -r -l ./batch-transfer-windows.zip ./* -x *.zip -x *.txt -x "keystore/*" && mv batch-transfer-windows.zip ../zip/ && cd ../..
# cd ./dist/mac && zip -r ./batch-transfer-mac.zip ./* -x *.zip -x *.txt -x "keystore/*" && mv batch-transfer-mac.zip ../zip/ && cd ../../
