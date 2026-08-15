VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "0.0.0-dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
BINARY  := bin/goblin

.PHONY: build build-all test clean

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/goblin

build-all:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/goblin-$(VERSION)-windows-amd64.exe ./cmd/goblin
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o bin/goblin-$(VERSION)-linux-amd64       ./cmd/goblin
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o bin/goblin-$(VERSION)-darwin-arm64       ./cmd/goblin
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o bin/goblin-$(VERSION)-darwin-amd64       ./cmd/goblin

test:
	go test ./...

clean:
	rm -f $(BINARY) bin/goblin-*
