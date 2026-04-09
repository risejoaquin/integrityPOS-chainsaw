BINARY=integritypos

build:
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc go build -o $(BINARY) ./cmd/server

run:
	go run ./cmd/server
