VERSION ?= dev

.PHONY: build build-relay build-web install clean fmt lint check

build:
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o tossit ./cmd/tossit

build-web:
	cd web && bun install && bun run build
	rm -rf internal/relay/web/dist
	cp -r web/dist internal/relay/web/dist

build-relay: build-web
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o relay ./cmd/relay

install: build
	cp tossit ~/.local/bin/tossit

clean:
	rm -f tossit relay
	rm -rf internal/relay/web/dist

fmt:
	gofmt -w .

lint:
	golangci-lint run ./...

check: fmt lint build build-relay
