GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOLANGCI_LINT=$(GOPATH)/bin/golangci-lint
GOLANGCI_LINT_RUN=$(GOLANGCI_LINT) run
GOVERALLS=$(GOPATH)/bin/goveralls

DIST=./dist
BINARY_NAME=treb
BINARY_NAME_WINDOWS=$(BINARY_NAME).exe

.PHONY: all restore build-prepare lint build build-unix build-windows test

all: build-prepare lint test build

build-prepare:
	mkdir -p $(DIST)
	go get -u github.com/golangci/golangci-lint/cmd/golangci-lint
	go get -u github.com/mattn/goveralls

lint: 
	$(GOLANGCI_LINT_RUN) ./...

test:
	$(GOTEST) -v -cover -coverprofile=./coverage.out ./...
	$(GOVERALLS) -coverprofile="./coverage.out" -service=travis-ci -repotoken $(COVERALLS_TOKEN)

build: build-unix build-windows

build-unix: build-prepare
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags "-X github.com/hylandsoftware/trebuchet/cmd.version=$(VERSION)" -o $(DIST)/$(BINARY_NAME) -v main.go
build-windows: build-prepare
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) -ldflags "-X github.com/hylandsoftware/trebuchet/cmd.version=$(VERSION)" -o $(DIST)/$(BINARY_NAME_WINDOWS) -v main.go