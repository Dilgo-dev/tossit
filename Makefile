VERSION ?= dev

.PHONY: build build-relay install clean fmt lint check

build:
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o tossit ./cmd/tossit

build-relay:
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o relay ./cmd/relay

install: build
	cp tossit ~/.local/bin/tossit

clean:
	rm -f tossit relay

fmt:
	gofmt -w .

lint:
	golangci-lint run ./...

check: fmt lint build build-relay
