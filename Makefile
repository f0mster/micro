BIN=./bin
GEN_FILENAME=rendall-rpc-code-gen
GEN_FILENAME_WIN=${GEN_FILENAME}.exe

VERSION=$(shell git describe --tags --abbrev=0)

build-linux: clear-bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -v -ldflags "-X main.version=${VERSION}" -o ${BIN}/${GEN_FILENAME} ./cmd/rendall-rpc-code-gen

install-linux: clear-bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go install -v -ldflags "-X main.version=${VERSION}" ./cmd/rendall-rpc-code-gen

build-win:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -v -ldflags "-X main.version=${VERSION}" -o ${BIN}/${GEN_FILENAME_WIN} ./cmd/rendall-rpc-code-gen

build-darwin:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -v -ldflags "-X main.version=${VERSION}" -o ${BIN}/${GEN_FILENAME} ./cmd/rendall-rpc-code-gen

install-darwin:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go install -v -ldflags "-X main.version=${VERSION}" ./cmd/rendall-rpc-code-gen

.PHONY: clear-bin
clear-bin:
	rm -f ./bin/rendall*

.PHONY: generate
generate:
	go generate ./...
	go vet ./...

.PHONY: format
format:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test:
	go test -v ./...

.DEFAULT_GOAL := build-darwin
