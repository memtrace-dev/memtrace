.PHONY: build test lint clean install

BINARY   = memtrace
VERSION ?= 0.1.0
LDFLAGS  = -ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/memtrace

test:
	go test ./... -v -count=1

lint:
	go vet ./...

clean:
	rm -rf bin/

install: build
	cp bin/$(BINARY) $(shell go env GOPATH)/bin/$(BINARY)

tidy:
	go mod tidy
