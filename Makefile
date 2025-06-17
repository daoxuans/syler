default: build

buildarm64:
	-rm ./bin/syler.arm64
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ./bin/syler.arm64 ./cmds/syler/main.go
	-upx -9 ./bin/syler.arm64

build:
	-rm ./bin/syler
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/syler ./cmds/syler/main.go
	-upx -9 ./bin/syler

clean:
	-rm ./bin -rf
